package watcher

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/sholdee/crd-schema-publisher/extractor"
	"github.com/sholdee/crd-schema-publisher/index"
	"github.com/sholdee/crd-schema-publisher/publisher"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

// Config holds the configuration for the CRD watcher.
type Config struct {
	Client     *apiextensionsclient.Clientset
	KubeConfig *rest.Config
	OutputDir  string
	Publisher  *publisher.Publisher // nil = extract-only
	Debounce   time.Duration
	Namespace  string
	LeaseName  string
	PodName    string
	HealthPort string
}

// Run starts the watcher with leader election and health server.
func Run(ctx context.Context, cfg Config) error {
	// Start health server before leader election
	healthReady := &atomic.Bool{}
	startHealthServer(cfg.HealthPort, healthReady)

	kubeClient, err := kubernetes.NewForConfig(cfg.KubeConfig)
	if err != nil {
		return fmt.Errorf("building kubernetes client: %w", err)
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      cfg.LeaseName,
			Namespace: cfg.Namespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: cfg.PodName,
		},
	}

	// Mark ready once we're participating in leader election
	healthReady.Store(true)

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		ReleaseOnCancel: true,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				log.Println("Acquired leadership, starting watch loop")
				runLeader(ctx, cfg)
			},
			OnStoppedLeading: func() {
				log.Println("Lost leadership, exiting")
				os.Exit(1)
			},
			OnNewLeader: func(identity string) {
				if identity != cfg.PodName {
					log.Printf("New leader elected: %s", identity)
				}
			},
		},
	})

	return nil
}

func runLeader(ctx context.Context, cfg Config) {
	// Initial publish on becoming leader
	log.Println("Running initial publish cycle")
	if err := publishCycle(cfg); err != nil {
		log.Printf("Initial publish failed: %v", err)
	}

	// Set up CRD informer
	trigger := make(chan struct{}, 1)
	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (k8sruntime.Object, error) {
			return cfg.Client.ApiextensionsV1().CustomResourceDefinitions().List(ctx, opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			return cfg.Client.ApiextensionsV1().CustomResourceDefinitions().Watch(ctx, opts)
		},
	}

	notify := cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { signalTrigger(trigger) },
		UpdateFunc: func(old, new interface{}) { signalTrigger(trigger) },
		DeleteFunc: func(obj interface{}) { signalTrigger(trigger) },
	}

	_, controller := cache.NewInformer(lw, &apiextensionsv1.CustomResourceDefinition{}, 0, notify)

	go controller.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), controller.HasSynced) {
		log.Println("Failed to sync informer cache")
		return
	}
	log.Println("CRD informer synced, watching for changes")

	debounceLoop(trigger, cfg.Debounce, func() error {
		return publishCycle(cfg)
	}, ctx.Done())
}

func signalTrigger(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
		// Channel already has a pending signal
	}
}

func debounceLoop(trigger <-chan struct{}, duration time.Duration, publish func() error, done <-chan struct{}) {
	var timer *time.Timer
	var timerC <-chan time.Time
	var publishing atomic.Bool

	for {
		select {
		case <-done:
			if timer != nil {
				timer.Stop()
			}
			return
		case <-trigger:
			if timer == nil {
				timer = time.NewTimer(duration)
				timerC = timer.C
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(duration)
			}
		case <-timerC:
			timer = nil
			timerC = nil
			if !publishing.CompareAndSwap(false, true) {
				log.Println("Publish already in progress, skipping")
				continue
			}
			go func() {
				defer publishing.Store(false)
				log.Println("Debounce timer fired, running publish cycle")
				if err := publish(); err != nil {
					log.Printf("Publish cycle failed: %v", err)
				}
			}()
		}
	}
}

func publishCycle(cfg Config) error {
	// Clean output dir
	if err := cleanDir(cfg.OutputDir); err != nil {
		return fmt.Errorf("cleaning output dir: %w", err)
	}

	// Extract
	crds, err := extractor.ListCRDs(cfg.Client)
	if err != nil {
		return fmt.Errorf("listing CRDs: %w", err)
	}
	if len(crds) == 0 {
		log.Println("No CRDs found")
		return nil
	}

	count, err := extractor.WriteSchemas(crds, cfg.OutputDir)
	if err != nil {
		return fmt.Errorf("writing schemas: %w", err)
	}
	log.Printf("Wrote %d schemas", count)

	// Generate index
	if err := index.Generate(cfg.OutputDir); err != nil {
		return fmt.Errorf("generating index: %w", err)
	}

	// Upload (if publisher configured)
	if cfg.Publisher != nil {
		if err := cfg.Publisher.Publish(cfg.OutputDir); err != nil {
			return fmt.Errorf("publishing: %w", err)
		}
	}

	log.Println("Publish cycle complete")
	return nil
}

func cleanDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(dir, 0o755)
		}
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func startHealthServer(port string, ready *atomic.Bool) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready.Load() {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("not ready"))
		}
	})
	go func() {
		server := &http.Server{Addr: ":" + port, Handler: mux}
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health server error: %v", err)
		}
	}()
}

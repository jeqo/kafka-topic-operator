package kafkatopic

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-logr/logr"
	kafkav1alpha1 "github.com/jeqo/kafka-topic-operator/pkg/apis/kafka/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const kafkaTopicFinalizer = "finalizer.topic.kafka.apache.org"

var log = logf.Log.WithName("controller_kafkatopic")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new KafkaTopic Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKafkaTopic{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("kafkatopic-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource KafkaTopic
	err = c.Watch(&source.Kind{Type: &kafkav1alpha1.KafkaTopic{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner KafkaTopic
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kafkav1alpha1.KafkaTopic{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileKafkaTopic implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKafkaTopic{}

// ReconcileKafkaTopic reconciles a KafkaTopic object
type ReconcileKafkaTopic struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a KafkaTopic object and makes changes based on the state read
// and what is in the KafkaTopic.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKafkaTopic) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling KafkaTopic")

	// Fetch the KafkaTopic instance
	kafkaTopic := &kafkav1alpha1.KafkaTopic{}
	err := r.client.Get(context.TODO(), request.NamespacedName, kafkaTopic)
	if err != nil {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	kafkaBootstrapServers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")

	// Initializing Kafka Admin Client
	admin, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": kafkaBootstrapServers})
	if err != nil {
		reqLogger.Error(err, "Failed to create Admin client: %s\n")
	}

	// defer admin.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	maxDur, err := time.ParseDuration("20s")
	if err != nil {
		return reconcile.Result{}, err
	}

	topicName := request.Name

	toBeDeleted := kafkaTopic.GetDeletionTimestamp() != nil
	if toBeDeleted {
		reqLogger.Info("KafkaTopic to be deleted")
		if contains(kafkaTopic.GetFinalizers(), kafkaTopicFinalizer) {
			results, err := admin.DeleteTopics(ctx, []string{topicName})
			reqLogger.Info("KafkaTopic to be deleted", "DeleteResults", results[0])
			if err != nil {
				return reconcile.Result{}, err
			}

			kafkaTopic.SetFinalizers(remove(kafkaTopic.GetFinalizers(), kafkaTopicFinalizer))
			err = r.client.Update(context.TODO(), kafkaTopic)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	if !contains(kafkaTopic.GetFinalizers(), kafkaTopicFinalizer) {
		reqLogger.Info("Setting KafkaTopic finalizer")
		if err := r.addFinalizer(reqLogger, kafkaTopic); err != nil {
			return reconcile.Result{}, err
		}
	}

	partitions := kafkaTopic.Spec.Partitions
	replicationFactor := kafkaTopic.Spec.ReplicationFactor
	topicConfig := kafkaTopic.Spec.TopicConfig

	// Check if Kafka Topic already exists
	metadata, err := admin.GetMetadata(&topicName, false, 20000)

	// if err != nil {
	// 	reqLogger.Error(err, "Error getting topic metadata")
	// 	return reconcile.Result{}, err
	// }

	topicMetadata := metadata.Topics[topicName]

	if topicMetadata.Error.Code() == kafka.ErrUnknownTopicOrPart {
		reqLogger.Info("Topic does not exist, creating a new Kafka Topic",
			"KafkaTopic.Partitions", partitions,
			"KafkaTopic.ReplicationFactor", replicationFactor,
			"KafkaTopic.Config", topicConfig)
		_, err := admin.CreateTopics(
			ctx,
			// Multiple topics can be created simultaneously
			// by providing more TopicSpecification structs here.
			[]kafka.TopicSpecification{{
				Topic:             topicName,
				NumPartitions:     partitions,
				ReplicationFactor: replicationFactor,
				Config:            topicConfig}},
			// Admin options
			kafka.SetAdminOperationTimeout(maxDur))

		if err != nil {
			reqLogger.Error(err, "Error creating topic")
			return reconcile.Result{}, err
		}

		// result := topicsCreated[0]
		// if result.Error.Code() != kafka.ErrNoError {
		// 	return reconcile.Result{}, metadata.Topics[topicName].Error
		// }
		return reconcile.Result{}, nil
	}

	resourceType, err := kafka.ResourceTypeFromString("TOPIC")
	results, err := admin.DescribeConfigs(ctx,
		[]kafka.ConfigResource{{Type: resourceType, Name: topicName}})

	if err != nil {
		reqLogger.Error(err, "Error describing topic")
		return reconcile.Result{}, err
	}

	result := results[0]
	// if result.Error.Code() != kafka.ErrNoError {
	// 	return reconcile.Result{}, metadata.Topics[topicName].Error
	// }

	// Check if config has changed
	currentConfig := make(map[string]string)
	changed := false

	// fmt.Printf("KafkaTopic config: %v\n", topicConfig)
	// fmt.Printf("KafkaTopic config on server: %v\n", result.Config)

	for _, entry := range result.Config {
		if len(topicConfig[entry.Name]) == 0 {
			currentConfig[entry.Name] = entry.Value
		} else {
			if topicConfig[entry.Name] != entry.Value {
				changed = true
				currentConfig[entry.Name] = topicConfig[entry.Name]
			} else {
				currentConfig[entry.Name] = entry.Value
			}
		}
	}

	reqLogger.Info("KafkaTopic configuration has changed", "KafkTopic.Changed", changed)
	if changed {
		config := make([]kafka.ConfigEntry, 0)
		for tc := range topicConfig {
			configEntry := kafka.ConfigEntry{Name: tc, Value: topicConfig[tc]}
			config = append(config, configEntry)
		}

		fmt.Printf("KafkaTopic config to be updated: %v\n", config)

		kafkaTopic.Spec.TopicConfig = currentConfig
		_, err := admin.AlterConfigs(ctx, []kafka.ConfigResource{{Type: resourceType, Name: topicName, Config: config}})
		if err != nil {
			return reconcile.Result{}, err
		}
		err = r.client.Update(context.TODO(), kafkaTopic)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileKafkaTopic) addFinalizer(reqLogger logr.Logger, m *kafkav1alpha1.KafkaTopic) error {
	reqLogger.Info("Adding finalizer to Kafka Topic")
	m.SetFinalizers(append(m.GetFinalizers(), kafkaTopicFinalizer))

	err := r.client.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update KafkaTopic with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}

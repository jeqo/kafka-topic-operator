# Kubernetes: Operator: Kafka Topic

## Build

```bash
go build -o $GOPATH/src/github.com/jeqo/kafka-topic-operator/build/_output/bin/kafka-topic-operator -gcflags all=-trimpath=$GOPAHT/src/github.com/jeqo -asmflags all=-trimpath=$GOPATH/src/github.com/jeqo github.com/jeqo/kafka-topic-operator/cmd/manager
```

```bash
docker build -t <IMAGE_NAME> -f build/Dockerfile .
```

Currently there is an issue with SDK:

```
operator-sdk build test
# github.com/jeqo/kafka-topic-operator/pkg/controller/kafkatopic
pkg/controller/kafkatopic/kafkatopic_controller.go:110:16: undefined: kafka.NewAdminClient
pkg/controller/kafkatopic/kafkatopic_controller.go:110:38: undefined: kafka.ConfigMap
pkg/controller/kafkatopic/kafkatopic_controller.go:166:35: undefined: kafka.ErrUnknownTopicOrPart
pkg/controller/kafkatopic/kafkatopic_controller.go:175:6: undefined: kafka.TopicSpecification
pkg/controller/kafkatopic/kafkatopic_controller.go:181:4: undefined: kafka.SetAdminOperationTimeout
pkg/controller/kafkatopic/kafkatopic_controller.go:195:23: undefined: kafka.ResourceTypeFromString
pkg/controller/kafkatopic/kafkatopic_controller.go:197:5: undefined: kafka.ConfigResource
pkg/controller/kafkatopic/kafkatopic_controller.go:231:20: undefined: kafka.ConfigEntry
pkg/controller/kafkatopic/kafkatopic_controller.go:233:19: undefined: kafka.ConfigEntry
pkg/controller/kafkatopic/kafkatopic_controller.go:240:39: undefined: kafka.ConfigResource
pkg/controller/kafkatopic/kafkatopic_controller.go:240:39: too many errors
```
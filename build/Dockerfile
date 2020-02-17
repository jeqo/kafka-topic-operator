FROM registry.access.redhat.com/ubi7/ubi:latest

ENV OPERATOR=/usr/local/bin/kafka-topic-operator \
    USER_UID=1001 \
    USER_NAME=kafka-topic-operator

COPY build/confluent.repo /etc/yum.repos.d/

RUN rpm --import https://packages.confluent.io/rpm/5.4/archive.key

RUN yum clean all && \
    yum install gcc cyrus-sasl-devel openssl-libs -y

RUN yum install librdkafka1 librdkafka-devel -y

# install operator binary
COPY build/_output/bin/kafka-topic-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

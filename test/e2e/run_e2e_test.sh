#!/bin/bash

print_version() {
    printf "tmctl version:\n---\n%s\n---\n" "`$TMCTL version`"
}

create_integration() {
    $TMCTL create broker e2e-test
    nohup $TMCTL watch > $WATCH_LOG &
    WATCH_PID=$!

    # watch command may take some time to start dumping logs
    sleep 5

    $TMCTL create target \
        --from-image docker.io/n3wscott/sockeye:v0.7.0 \
        --name e2e-sockeye
    $TMCTL create source httppoller \
        --name e2e-source \
        --endpoint https://www.githubstatus.com/api/v2/status.json \
        --eventType e2e-test-event \
        --interval 600s \
        --method GET
    $TMCTL create trigger \
        --target e2e-sockeye \
        --source e2e-source
    $TMCTL create transformation \
        --target e2e-sockeye <<EOF
data:
- operation: parse
  paths:
  - key: .
    value: JSON
- operation: store
  paths:
  - key: ~url
    value: page.url
- operation: delete
  paths:
  - key:
- operation: add
  paths:
  - key: URL
    value: ~url
EOF
}

send_event() {
    $TMCTL send-event '{"page": {"url":"https://www.githubstatus.com","source":"cli send-event"}}' --eventType e2e-test-event
}

validate_describe_output() {
    DESCRIBE_OUTPUT="`$TMCTL describe`"
    DESCRIBE_LEN=$(echo "$DESCRIBE_OUTPUT" | wc -l)
    if [ $DESCRIBE_LEN -ne 15 ]; then
        printf "Describe output len missmatch: expected 15, got %s\n" $DESCRIBE_LEN
        return 1
    fi

    OFFLINE_COMPONENTS=$(echo "$DESCRIBE_OUTPUT" | grep "offline" | sed -r '/^\s*$/d')
    if [ ! -z "$OFFLINE_COMPONENTS" ]; then
        printf "Some components are offline:\n%s\n" $OFFLINE_COMPONENTS
        return 1
    fi

    SOCKEYE_URL=$(echo "$DESCRIBE_OUTPUT" | \
        grep "e2e-sockeye" | \
        tail -1 | \
        awk '{print $(NF)}' | \
        awk -F '[()]' '{print $2}')
    
    SOCKEYE_RESPONSE=$(curl -sI $SOCKEYE_URL | head -1)
    # GitHub Action curl to localhost returns "HTTP/1.1 403 Forbidden"
    if [ $(echo "$SOCKEYE_RESPONSE" | grep -c 'HTTP/1.1 200 OK\|HTTP/1.1 403 Forbidden') -ne 1 ]; then
        printf "Unexpected sockeye service response: %s %s\n" $SOCKEYE_URL "$SOCKEYE_RESPONSE"
        return 1
    fi
}

validate_watch_log() {
    CONTROL_LINES=(
        '  source: local.e2e-source'
        '  type: e2e-test-transformation.output'
        '      "name": "GitHub",'
        '      "source": "cli send-event"'
        '    "URL": "https://www.githubstatus.com"'
    )

    for LINE in "${CONTROL_LINES[@]}"; do
        LINE_COUNTER=$(grep -c "$LINE" $WATCH_LOG)
        if [ $LINE_COUNTER -ne 1 ]; then
            printf "Watch validation: expected exactly 1 line \"%s\", got %s\n" "$LINE" $LINE_COUNTER
            return 1
        fi
    done
}

cleanup() {
    echo "Cleaning up test environment"
    kill -INT $WATCH_PID
    $TMCTL delete --broker e2e-test

    BROKERS="`$TMCTL brokers`"
    if [ ! -z "$BROKERS" ]; then
        printf "Unexpected \"tmctl brokers\" output: %s\n" "$BROKERS"
        exit 1
    fi
    
    rm -r $TESTDIR
}

TESTDIR=$(dirname "$0")/_run
TMCTL=$TESTDIR/tmctl_test
WATCH_LOG=$TESTDIR/watch-output.log
mkdir -p $TESTDIR

go build -o $TMCTL main.go

HOME=$TESTDIR

print_version
if [ $? -ne 0 ]; then 
    cleanup
    printf "\"tmctl version\" validation failed\n"
    exit 1
fi

create_integration
if [ $? -ne 0 ]; then 
    cleanup
    printf "\"tmctl create\" validation failed\n"
    exit 1
fi

sleep 3

send_event
if [ $? -ne 0 ]; then 
    cleanup
    printf "\"tmctl send-event\" validation failed\n"
    exit 1
fi

validate_describe_output
if [ $? -ne 0 ]; then 
    cleanup
    printf "\"tmctl describe\" output validation failed\n"
    exit 1
fi

validate_watch_log
if [ $? -ne 0 ]; then 
    cleanup
    printf "\"tmctl watch\" output validation failed\n"
    exit 1
fi

cleanup

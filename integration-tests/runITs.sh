#!/bin/bash

echo "RDB OPERATOR component integration test"
docker-compose up -d
echo "Docker containers for ITs ready."
sleep 5
sh prepareTests.sh
echo "Preparation to tests OK"
echo "Integration tests start"
PROFILE=dev go test xqledger/rdboperator/apilogger -v 2>&1 | go-junit-report > ../testreports/apilogger.xml
PROFILE=dev go test xqledger/rdboperator/configuration -v 2>&1 | go-junit-report > ../testreports/configuration.xml
PROFILE=dev go test xqledger/rdboperator/utils -v 2>&1 | go-junit-report > ../testreports/utils.xml
PROFILE=dev go test xqledger/rdboperator/mongodb -v 2>&1 | go-junit-report > ../testreports/mongodb.xml
PROFILE=dev go test xqledger/rdboperator/kafka -v 2>&1 | go-junit-report > ../testreports/kafka.xml
echo "Integration tests complete"
echo "Cleaning up..."
cd ../integration-tests
docker-compose down
echo "Clean up complete. Bye!"
# TODO fail process in case of failed tests
# TODO export result to report for pipeline

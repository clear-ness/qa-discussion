#!/bin/bash

goose -dir ./db/migrations mysql "root:@tcp(localhost:3306)/qa_discussion_test?parseTime=true" $1

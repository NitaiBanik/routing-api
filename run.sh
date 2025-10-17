#!/bin/bash

# Kill any existing processes on port 3000
lsof -ti:3000 | xargs kill -9
sleep 1

# Start the routing API
echo "Starting routing API on port 3000..."
go run cmd/server/main.go

# Routing API

A simple Go HTTP routing API that implements round-robin load balancing for distributing requests across multiple application API instances.

## Getting started

You'll need Go 1.21 or newer installed on your machine.

1. **Clone this repo:**
   ```bash
   git clone <your-repo-url>
   cd routing-api
   ```

2. **Get the dependencies:**
   ```bash
   go mod tidy
   ```

3. **Run it:**
   ```bash
   go run main.go
   ```

4. **Test the routing:**
   ```bash
   curl -X POST http://localhost:3000/testapi \
     -H "Content-Type: application/json" \
     -d '{"request_number": 1, "message": "Hello World"}'
   ```

### Quick start with testing
```bash
./run.sh
```

### Build it
```bash
go build -o routing-api main.go
```

### Run tests
```bash
go test ./...
```

Run specific tests:
```bash
go test ./handlers   
go test -run TestAPI
```

## How the service works

### POST /testapi (or any path)

Send it any valid JSON and it'll forward the request to one of the configured application APIs using round-robin load balancing.

```bash
curl -X POST http://localhost:3000/testapi \
  -H "Content-Type: application/json" \
  -d '{"key": "value", "number": 42, "array": [1, 2, 3]}'
```

The request will be forwarded to one of the application APIs and you'll get back their response.

### Testing Round-Robin Load Balancing

To see the round-robin behavior in action, send multiple requests:

```bash
curl -X POST http://localhost:3000/testapi -H "Content-Type: application/json" -d '{"request_number": 1, "message": "test 1"}'
curl -X POST http://localhost:3000/testapi -H "Content-Type: application/json" -d '{"request_number": 2, "message": "test 2"}'
curl -X POST http://localhost:3000/testapi -H "Content-Type: application/json" -d '{"request_number": 3, "message": "test 3"}'
```

Each request will be distributed to different application API instances in round-robin fashion.

### GET /health

Check if the routing service is alive and well:

```bash
curl http://localhost:3000/health
```

```json
{
  "status": "healthy",
  "service": "routing-api",
  "timestamp": "2024-01-01T12:00:00Z"
}
```


## Configuration

Copy the example env file and modify it:
```bash
cp env.example .env
# edit .env with your values
```

Or set them directly:
```bash
PORT=3000 go run main.go
```

Configure application APIs via environment variables:
```bash
API_1=http://localhost:8080
API_2=http://localhost:8081
API_3=http://localhost:8082
```

## Project structure

```
routing-api/
├── main.go              # Where it all starts
├── go.mod               # Dependencies
├── env.example          # Environment variables template
├── run.sh               # Script to run and test the API
├── config/
│   └── config.go        # Handles environment variables
├── handlers/
│   └── handlers.go      # The actual API endpoints
├── middleware/
│   └── middleware.go    # Logging and other middleware
└── README.md            # This file
```
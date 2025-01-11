# Git Security Scanner

A Go-based security monitoring service that automatically scans Git repositories for vulnerabilities using osv-scanner.

## Features

### Planned Features

- Periodic scanning of Git repositories using osv-scanner
- Automatic vulnerability detection for new commits
- REST API endpoints for retrieving scan results
- Basic vulnerability reporting and statistics

Tools to use : Testify for testing & sambler for function

## Overview

Our RBAC (Role-Based Access Control) system uses Authentik with RESgate for authentication and authorization across microservices. This setup provides a simple but flexible permission system that scales with new services.

# API Authentication & Authorization Documentation

## Architecture

```
Client → RESgate → Authentik Proxy → Microservices
```

## Permission Structure

### Basic Format

Every service follows this permission pattern:

```
read:service-name    # For GET operations
write:service-name   # For POST/PUT/DELETE operations
admin:service-name   # For full access (optional)
```

### Special Permissions (when needed)

```
approve:service-name         # For specific actions
read:service-name:type      # For specific resource types
```

## Public Endpoints

### Basic Configuration

Every service can define public endpoints that bypass authentication:

```yaml
service_name: your-service-name
permissions:
  - name: read:your-service
    description: "Read access to your service"
  - name: write:your-service
    description: "Write access to your service"
public_endpoints:
  - path: "/api/v1/public/*"
    methods: ["GET"]
  - path: "/api/v1/status"
    methods: ["GET"]
```

### Gateway Configuration

RESgate needs specific configuration for public endpoints:

```yaml
paths:
  "/api/v1/public/*":
    disable_auth: true
  "/api/v1/status":
    disable_auth: true
```

### Authentik Proxy Setup

Configure Authentik to bypass authentication for public paths:

```yaml
proxy:
  rules:
    - name: "Public Endpoints"
      paths:
        - "/api/v1/public/*"
        - "/api/v1/status"
      policy: bypass_authentication
```

## Service Setup

### 1. Create permissions.yaml

Every service must have a permissions.yaml file:

```yaml
service_name: your-service-name
permissions:
  - name: read:your-service
    description: "Read access to your service"
  - name: write:your-service
    description: "Write access to your service"
  # Add special permissions if needed
  - name: approve:your-service
    description: "Approval rights for your service"
public_endpoints:
  - path: "/api/v1/public/status"
    methods: ["GET"]
    rate_limit: "1000/hour"
```

### 2. Service Registration

Services automatically register their permissions on startup.

## Group Management

### Pattern-Based Groups

Main groups are configured once with patterns:

```yaml
groups:
  System Admins:
    pattern: "*:*" # Full access to everything
  Service Operators:
    pattern: "read:*, write:*" # Read/write to everything
  Viewers:
    pattern: "read:*" # Read-only access to everything
```

### Special Groups

For specific business needs:

```yaml
Sales Team:
  patterns:
    - "read:orders"
    - "write:orders"
    - "read:products"
```

## Implementation Guide

### 1. New Service Setup

1. Create `permissions.yaml`:

```yaml
service_name: orders-service
permissions:
  - name: read:orders
    description: "Read access to orders"
  - name: write:orders
    description: "Write access to orders"
  - name: approve:orders:high-value
    description: "Approve high-value orders"
public_endpoints:
  - path: "/api/v1/public/orders/status"
    methods: ["GET"]
    rate_limit: "100/hour"
```

2. Configure RESgate:

```yaml
paths:
  "/api/v1/orders/*":
    auth_required: true
  "/api/v1/public/orders/status":
    disable_auth: true
```

3. Configure Authentik proxy:

```yaml
proxy:
  rules:
    - name: "Orders Service"
      paths:
        - "/api/v1/orders/*"
      policy: require_authentication
    - name: "Public Status"
      paths:
        - "/api/v1/public/orders/status"
      policy: bypass_authentication
```

### 2. Group Configuration

```yaml
groups:
  Order Managers:
    patterns:
      - "read:orders"
      - "write:orders"
      - "approve:orders:*"
  Order Viewers:
    patterns:
      - "read:orders"
```

## Examples

### 1. Basic Service

```yaml
# permissions.yaml
service_name: product-catalog
permissions:
  - name: read:products
    description: "Read product information"
  - name: write:products
    description: "Modify product information"
public_endpoints:
  - path: "/api/v1/public/products"
    methods: ["GET"]
    rate_limit: "1000/hour"
```

### 2. Complex Service

```yaml
# permissions.yaml
service_name: order-management
permissions:
  - name: read:orders
    description: "View orders"
  - name: write:orders
    description: "Create/modify orders"
  - name: approve:orders:high-value
    description: "Approve high-value orders"
  - name: admin:orders
    description: "Full order management access"
public_endpoints:
  - path: "/api/v1/public/orders/tracking"
    methods: ["GET"]
    rate_limit: "500/hour"
  - path: "/api/v1/public/orders/status"
    methods: ["GET"]
    rate_limit: "1000/hour"
```

# OpenLambda Development Environment Setup Guide

## Overview
This document explains how to set up a development environment with:

- **1 Boss process** (management/control plane)
- **1+ Worker processes** (execution environment) that can be scaled up/down

## 1. Build the OL Minimal Image

First, create the base container image that workers will use:

```bash
# From the open-lambda root directory
make ol imgs/ol-min
```

This builds a minimal container image (`ol-min`) that workers use to execute Lambda functions.

## 2. Start the Boss Process

The **boss** manages worker scaling and task distribution.

```bash
./ol boss up
```

### What this does:
- Starts the boss on port **5000** (default)
- Creates default configs in **boss/config.json**
- Waits for scaling requests

## 3. Scale Workers (1, 2, 3, etc.)

Workers are dynamically created via the boss's scaling API.

### Start 1 Worker
```bash
curl -X POST http://localhost:5000/scaling/worker_count -d "1"
```
- Worker starts on port **6000** (default)
- Worker config is loaded from **template.json**

### Scale Up to 2 Workers
```bash
curl -X POST http://localhost:5000/scaling/worker_count -d "2"
```
- A **second worker** starts on port **6001**
- Each new worker increments the port number

## 4. Shutting Down
When the boss receives a kill signal, it will automatically clean up the workers as well.


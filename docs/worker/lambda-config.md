# Lambda Configuration Guide

This document provides an overview of per-lambda configuration in OpenLambda, detailing trigger types, configuration options, and usage.

## 1. Overview
Each lambda function in OpenLambda can optionally be configured using an `ol.yaml` file located in the function's directory. This configuration file defines how the function is triggered and any specific execution parameters.

## 2. Configuration Structure
A typical `ol.yaml` file follows this structure:

```yaml
triggers:
  http:
    - method: PUT
    - method: PATCH

environment:
  MY_ENV_VAR1: "value1"
  MY_ENV_VAR2: "value2"
```

## 3. Configuration Options

### a. Triggers
OpenLambda only supports HTTP trigger for now, but future development plans include supporting other trigger types.

#### HTTP Triggers
Defines which HTTP methods can be used to invoke the lambda.

Example:
```yaml
triggers:
  http:
    - method: GET
    - method: POST
```
In this case, the lambda accepts GET and POST requests.

### b. Environment Variables
Defines environment variables that will be available to the lambda function at runtime.

Example:
```yaml
environment:
  MY_ENV_VAR1: "production"
  MY_ENV_VAR2: "enabled"
```

These variables can be accessed in your lambda code using standard environment variable methods (e.g., `os.environ` in Python).

**Note:** Environment variables defined in `ol.yaml` are written to a `.env` file in the lambda's directory during execution. If your lambda already has a `.env` file, it will be overwritten with the values from `ol.yaml`.

## 4. How to Use
### a. Define Configuration
Create an `ol.yaml` file inside the lambda function directory with the desired configuration.

### b. Deploy and Run
Ensure the OpenLambda framework is set up and execute your lambda as per the defined triggers.

## 5. Validations
- HTTP triggers must specify valid HTTP methods (GET, POST, PUT, DELETE, etc.).
- If no triggers are specified or no configuration file exists in the lambda function directory, OpenLambda will apply default behavior allowing all HTTP methods.

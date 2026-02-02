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

### c. Special Environment Variables

#### OL_ENTRY_FILE
By default, OpenLambda expects Python lambda functions to be defined in a file named `f.py`. You can override this by setting the `OL_ENTRY_FILE` environment variable to specify a different entry file.

Example:
```yaml
environment:
  OL_ENTRY_FILE: "app.py"
```

With this configuration:
- OpenLambda will look for `app.py` instead of `f.py` when detecting the Python runtime
- The Python runtime will import the `app` module instead of `f`
- For standard functions, define your handler as `def f(event)` in the specified file
- For Flask/WSGI applications, define your `app` object in the specified file

This is useful when you want to use conventional naming (e.g., `app.py` for Flask applications) or integrate existing code without renaming files.

### d. Sandbox Reuse

#### reuse_sandbox
By default, OpenLambda reuses the same sandbox across multiple invocations of a lambda function to improve performance. In some cases, such as when strict isolation is required or when avoiding state persistence between invocations, it may be desirable to create a fresh sandbox for each invocation.

This behavior can be controlled using the reuse-sandbox option.

Example:
```yaml
reuse-sandbox: false
```

With this configuration:

- A new sandbox is created for each lambda invocation
- The sandbox is destroyed after the invocation completes

If reuse-sandbox is not specified, OpenLambda defaults to reusing sandboxes across invocations.

## 4. How to Use
### a. Define Configuration
Create an `ol.yaml` file inside the lambda function directory with the desired configuration.

### b. Deploy and Run
Ensure the OpenLambda framework is set up and execute your lambda as per the defined triggers.

## 5. Validations
- HTTP triggers must specify valid HTTP methods (GET, POST, PUT, DELETE, etc.).
- If no triggers are specified or no configuration file exists in the lambda function directory, OpenLambda will apply default behavior allowing all HTTP methods.

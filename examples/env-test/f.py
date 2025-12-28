import os
import json

def f(event):
    """
    Lambda function that demonstrates environment variable usage.
    Returns all environment variables that were configured in ol.yaml
    """
    
    # Get environment variables from config
    env_vars = {
        "MY_ENV_VAR": os.environ.get("MY_ENV_VAR", "not set"),
        "DATABASE_URL": os.environ.get("DATABASE_URL", "not set"),
        "DEBUG_MODE": os.environ.get("DEBUG_MODE", "not set"),
        "API_KEY": os.environ.get("API_KEY", "not set"),
        "CUSTOM_PATH": os.environ.get("CUSTOM_PATH", "not set"),
    }
    
    response = {
        "message": "Environment variables test",
        "event": event,
        "configured_env_vars": env_vars,
        "all_env_vars_count": len(os.environ),
    }
    
    # If debug mode is enabled, show all environment variables
    if os.environ.get("DEBUG_MODE") == "true":
        response["all_env_vars"] = dict(os.environ)
    
    return response
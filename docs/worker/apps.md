# Deploying Applications

## Agricultural Forecasting API (FastAPI)

[ag_forecasting_api](https://github.com/UW-Madison-DSI/ag_forecasting_api) is a FastAPI application that provides crop disease forecasting models for corn and soybean in Wisconsin, developed by University of Wisconsin-Madison plant pathology experts.

Initialize a worker with the min image:

```bash
./ol worker init -i ol-min
```

Edit `myworker/config.json` to increase memory limit (512MB needed for this app):

```json
"limits": {
    "mem_mb": 512,
    ...
}
```

Start the worker:

```bash
./ol worker up -d
```

Create `ol.yaml` to configure the app for OpenLambda:

```yaml
triggers:
  http:
    - method: "*"
environment:
  OL_ENTRY_FILE: app.py
  MEASUREMENTS_CACHE_DIR: /host/tmp/cache
```

Install pip-compile and pin requirements.txt to versions suitable for OpenLambda:

```bash
./ol admin install examples/pip-compile
curl -X POST -d 'https://raw.githubusercontent.com/tylerharter/ag_forecasting_api/main/requirements.txt' http://localhost:5000/run/pip-compile/url > requirements.txt
```

Install and test:

```bash
./ol admin install -c ol.yaml -r requirements.txt https://github.com/tylerharter/ag_forecasting_api.git

# simple test
curl http://localhost:5000/run/ag_forecasting_api/
```

NOTE: the full app doesn't work yet (we need to make sure code and writable directories are as expected).  This fails:

```bash
# get a forecast for the ALTN station
curl "http://localhost:5000/run/ag_forecasting_api/ag_models_wrappers/wisconet?forecasting_date=2024-07-01&risk_days=1&station_id=ALTN"
```

TODO: update ag_forecasting_api URLs from tylerharter fork to UW-Madison-DSI once env option is merged upstream.

## TODO: add more example apps

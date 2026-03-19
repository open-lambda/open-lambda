1# Deploying Applications

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
  OL_ASGI_ENTRY: app
  MEASUREMENTS_CACHE_DIR: /host/tmp/cache
  STATIONS_CACHE_FILE: /host/tmp/cache/wisconsin_stations_cache.csv
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

# get a forecast for the ALTN station
curl "http://localhost:5000/run/ag_forecasting_api/ag_models_wrappers/wisconet?forecasting_date=2024-07-01&risk_days=1&station_id=ALTN"
```

Note, the first request may take minutes because OpenLambda will install all the packages in requirements.txt upon the first call.

TODO: update ag_forecasting_api URLs from tylerharter fork to UW-Madison-DSI once env option is merged upstream.

## Global Mosquito Observations Dashboard API (Flask + MySQL)

[Global Mosquito Observations Dashboard](https://github.com/UW-Madison-DSI/Global-Mosquito-Observations-Dashboard), developed by the UW-Madison Data Science Institute (DSI), is a Flask + MySQL application that aggregates and displays mosquito observation data from different citizen science data sources.  

### Installation

1. Clone the repo:
   ```bash
   git clone https://github.com/UW-Madison-DSI/Global-Mosquito-Observations-Dashboard.git
   ```
2. Edit `docker-compose.yml`:
```yml
services:
  db:
    image: mysql:latest
    environment:
      MYSQL_DATABASE: mosquito_dashboard
      MYSQL_USER: webuser
      MYSQL_PASSWORD: password
      MYSQL_ROOT_PASSWORD: root
      MYSQL_ALLOW_EMPTY_PASSWORD: 1
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${MYSQL_ROOT_PASSWORD}"]
      interval: 10s
      timeout: 5s
      retries: 3
    ports:
      - "3306:3306"
    volumes:
      - ./database:/docker-entrypoint-initdb.d
      - ./mysql:/var/lib/mysql
    networks:
      - network

networks:
  network:
    driver: bridge
```
3. Start the database:
```bash
docker compose up
```

4. Initialize a worker:

```bash
./ol worker init -i ol-min
```

5. Start the worker:

```bash
./ol worker up -d
```

6. Create `ol.yaml` to configure the app for OpenLambda:

```yaml
triggers:
  http:
    - method: "*"
environment:
  OL_ENTRY_FILE: "app.py"
  FLASK_ENV: "development"
  DB_HOST: "127.0.0.1"
  DB_PORT: "3306"
  DB_USERNAME: "webuser"
  DB_DATABASE: "mosquito_dashboard"
  DB_PASSWORD: "password"
```

7. Install pip-compile and pin requirements.txt to versions suitable for OpenLambda:

```bash
./ol admin install examples/pip-compile
curl -X POST -d 'https://raw.githubusercontent.com/UW-Madison-DSI/Global-Mosquito-Observations-Dashboard/refs/heads/main/src/server/requirements.txt' http://localhost:5000/run/pip-compile/url > mosquito_requirements.txt
```

8. Install and test:

```bash
/ol admin install -c ol.yaml -r mosquito_requirements ./Global-Mosquito-Observations-Dashboard/server

# simple test
curl http://localhost:5000/run/server/

# get a forecast for the ALTN station
curl -X GET "http://localhost:5000/run/server/observations/habitat-mapper"
```
   
    

## TODO: add more example apps

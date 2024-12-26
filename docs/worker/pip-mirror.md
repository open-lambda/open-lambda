# Installing a Pip Mirror

When running python based lambdas in OpenLambda packages
are installed with pip. By default, pip installs packages
through an API to pypi.org, which cannot always be trusted.

Pypi.org cannot guarantee that every indexed package is safe,
thus a local mirrored index for pip may be beneficial.

### Creating and Defining the Bandersnatch Mirror

In order to setup a bandersnatch pip mirror first install bandersnatch with:
```
pip install bandersnatch
```

Once you have installled bandersnatch, you will have to create the config file.
To do this run:
```
bandersnatch mirror
```

This will create a bandersnatch.conf file. In this configuration
file you can set different settings for the mirror.

Furthermore, you can decide the backend for the bandersnatch mirror.
This can be a docker container, web server (such as nginx), or a filesystem.

OpenLambda only supports a docker container or web server.

For documentation on how to set up the mirror use: [bandersnatch docs](https://bandersnatch.readthedocs.io/en/latest/mirror_configuration.html)

It is important to note that if you do not use an allowlist or 
blacklist for configuration you will be installing the whole 
pypi index which is several terabytes.

When deciding which packages to install into the mirror it is important
to note that the current implementation of the package puller, when
a pip mirror is defined, is to not use pypi.org even upon install
failure.

Furthermore, since OpenLambda does not install dependencies with 
pip's `--no-deps` please ensure that any packages you wish to 
install have all required dependencies
within the mirror as well. For more information on how OpenLambda
installs packages view: [pypi-packages](pypi-packages.md). 

Once you have set up the bandersnatch mirror and are happy with the
configuration file you can run `bandersnatch mirror` once more to 
begin installation. This may take several minutes.

### Nginx Support

One possible way to serve the pip mirror is with a web server.

To setup a webserver with nginx first run:

```
sudo apt update
sudo apt install nginx
```

Create a configuration file in `/etc/nginx/sites-available/`,
a starting configuration is:
```
server {
            listen {ip}:{port};
            server_name name;  # Change this to your server's hostname

            # Serve static files from the bandersnatch mirror directory
            location /pypi/ {
                alias {mirror/web};  # This should point to the bandersnatch web directory
                autoindex on;           # Enables directory listing
            }
}
```

Edit the location section of this file such that you have:
```
location /pypi/ {
    alias {path to pip mirror with /web/ at the end};
    autoindex on;
}
```

Note `/pypi/` can be changed as you wish, this will be reflected in the url.

Additionally you will have to use an ip as localhost does not map to the same ip
within OpenLambda. Thus you will have to change the listen field to:

```
listen {internal/external ip address}:{port}
```

If you running OpenLambda on the same network as the web server you will have
to use the internal ip. Otherwise you will have to use an external ip.

Create a link to the conf file within the sites enabled directory with:
```
ln -s /etc/nginx/sites-available/{conf file} /etc/nginx/sites-enabled/
```

After you have finished editing these fields test your configuration with:
```
sudo nginx -t
```

Then start the web server with:
```
sudo systemctl reload nginx
```

To verify the web server is correctly connected to your mirror use:
```
curl http://{ip}/pypi/ 
```

If you get an error 403, this is likely due to read permissions for your
bandersnatch mirror. As nginx requires read and execute access, additionally
nginx may want the user `www-data` to have ownership. To do this use:
```
sudo chmod -R 755 {path to mirror} # for read permissions
sudo chown -R www-data:www-data {path to mirror} # for ownership
```

### Worker Setup

After the mirror has finished installation and you have set up a method to serve the mirror, 
you should initialize a worker with:
```
ol worker init -p worker_name -i base_image
```
You can choose the worker name and the base image to initialize.

Then in `config.json` edit the `pip_mirror` field with the url to your pip mirror.
Do not include simple at the end, and ensure that it is the url with the location field.
`{url}/{location}`.

Before starting the worker ensure that in the registry directory
you have created a `requirements.in` file with the required packages.
Then get the required dependencies with:
```
pip-compile /path/to/worker/registry/requirements.in
```

If you do not yet have `pip-compile` run:
```
pip install pip-tools
```
This will create a `requirements.txt` which contains all packages that
will be installed.

Then you can start the worker with:
```
ol worker up
```

This will not start any installs until you have sent it a 
POST request. 

### Potential Issues

If you run into an error it will likely present as unable to find
/host/files. This is because the install failed leading to the target
directory for packages to be undefined. 

Thus some possible causes may be:
1. Package requested is not in the mirror.
2. Incorrect pip mirror url or directory. This may be due to read access for the mirror or web directory.
3. Packages not including version numbers.
4. Updates in requirements for a package causing the mirror to become outdated.

To better analyze this the pip install command has `-vvv` in order to help debugging.

Additionally, installs do not yet account for platform specific dependencies or
conditional dependencies in general.

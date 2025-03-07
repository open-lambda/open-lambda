# Installing a Pip Mirror

When running python-based lambdas in OpenLambda packages
are installed with pip. By default, pip installs packages
through an API to pypi.org, which cannot always be trusted.

Pypi.org cannot guarantee that every indexed package is safe,
thus a local mirrored index for pip allows the user to control
what packages are installed ensuring a more secure environment.

Additionally, a pip mirror allows for offline installs which 
are useful when the machine running OpenLambda does not have
internet access or when pypi.org is down.

Furthermore, installation of packages from a pip mirror is faster
as the packages are downloaded in parallel and do not require network 
transactions.

### Creating and Defining the Bandersnatch Mirror

Install bandersnatch with:
```
pip install bandersnatch
```

Create the config file:
```
bandersnatch mirror
```

Furthermore, you can decide the backend for the bandersnatch mirror.
This can be a docker container, web server (such as nginx), or a filesystem.

OpenLambda only supports a docker container or web server.

For documentation on how to set up the mirror use: [bandersnatch docs](https://bandersnatch.readthedocs.io/en/latest/mirror_configuration.html)

It is important to note that if you do not use an allowlist or 
blacklist for configuration you will be installing the whole 
pypi index which is several terabytes.


An example bandersnatch.conf file is:
```
[mirror]
; Storage directory (where the mirror will be stored)
directory = /path/to/nginx
; Upstream repo
master = https://pypi.org
; Number of workers
workers = 3
; Request timeout (sec)
timeout = 15
; Global timeout (sec)
global-timeout = 18000

[plugins]
; Enabled plugins
enabled =
    project_requirements
    project_requirements_pinned

[allowlist]
; Directory where the requirements file is located
requirements_path = /path/to/requirements_file
; Name of the requirements file
requirements = requirements.txt
```
To create the requirements file read: [pypi-packages](pypi-packages.md).

Furthermore, since OpenLambda does not install dependencies with 
pip's `--no-deps` please ensure that any packages you wish to 
install have all required dependencies
within the mirror as well.

Run the following command to begin installation:
```
bandersnatch mirror
```

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
    alias {path to pip mirror with /web/simple/ at the end};
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
you should initialize a worker. Check: [getting-started](getting-started.md) for more information.

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

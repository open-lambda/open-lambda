Running Open Lambda in a virtual machine is the easiest way to see it in action. Just make sure [Vagrant](https://www.vagrantup.com/) and [Ansible](https://www.ansible.com/) are installed. The whole install process will take around 15 minutes as a virtual machine, Docker containers and packages need to be downloaded and setup.

Simply type `vagrant up` and grab a cup of coffee. `vagrant ssh` will drop you into the virtual machine. Open Lambda lives in /opt/open-lambda. When you are done either type `vagrant halt` or `vagrant destroy`.

## Notes
* Ansible prints out the stdout only after a command has returned. It is working even if it doesn't look like it.
* In the future, a shell script will be used so that Ansible isn't required.
* Currently until Open Lambda is public it grabs your ssh keys to pull from the repo. If this step fails, just enter the virtual machine and clone the repo.

Running Open Lambda in a virtual machine is the easiest way to see it in action. Just make sure [Vagrant](https://www.vagrantup.com/) installed along with a backend such as VirtualBox.

Setting up the run time environment can be done with a shell script or [Ansible](https://www.ansible.com/) The whole install process can take a few minutes as a virtual machine, Docker containers and packages need to be downloaded and setup.

If you choose Ansible to setup, change the provision settings in the Vagrantfile. Simply type `vagrant up`, grab a cup of coffee, and then go to http://localhost:8080 to try out a scalable chat application. When you are done either type `vagrant halt` or `vagrant destroy`.

## Notes
* Ansible prints out the stdout only after a command has returned. It is working even if it doesn't look like it.
* `vagrant ssh` will drop you into the virtual machine. Open Lambda lives in /opt/open-lambda if it is installed for you.

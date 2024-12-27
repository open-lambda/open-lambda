# OpenLambda

[![CI](https://github.com/open-lambda/open-lambda/actions/workflows/ci.yml/badge.svg)](https://github.com/open-lambda/open-lambda/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

OpenLambda is an Apache-licensed serverless computing project, written
(mostly) in Go and based on Linux containers.  The primary goal of
OpenLambda is to enable exploration of new approaches to serverless
computing.  We hope to eventually make it suitable for use in
production as well.

The main system implemented so far is a single-node OpenLambda worker
that can take HTTP requests and invoke lambdas locally to compute
responses.

You can read more about the **OpenLambda worker** [here](docs/worker/README.md) or just get started
by [deploying a worker](docs/worker/getting-started.md).

We are currently working on a cluster mode, where a pool of VMs
running the worker service are managed by a centralized **OpenLambda
boss**.  With a bit of work, you could also manually deploy workers
yourself and put an HTTP load balancer in front of them.

## Related Publications

* [Forklift: Fitting Zygote Trees for Faster Package Initialization](https://dl.acm.org/doi/pdf/10.1145/3702634.3702952) by Yang <i>et al.</i> (WoSC '24)
* [SOCK: Rapid Task Provisioning with Serverless-Optimized Containers](https://www.usenix.org/system/files/conference/atc18/atc18-oakes.pdf) by Oakes <i>et al.</i> (ATC '18)
* [Pipsqueak: Lean Lambdas with Large Libraries](https://ieeexplore.ieee.org/document/7979853) by Oakes <i>et al.</i> (ICDCSW '17)
* [Serverless Computation with OpenLambda](https://www.usenix.org/system/files/login/articles/login_winter16_03_hendrickson.pdf) by Hendrickson <i>et al.</i> (;login '16)
* [Serverless Computation with OpenLambda](https://www.usenix.org/system/files/conference/hotcloud16/hotcloud16_hendrickson.pdf) by Hendrickson <i>et al.</i> (HotCloud '16)

## License

This project is licensed under the Apache License - see the [LICENSE.md](LICENSE.md) file for details.

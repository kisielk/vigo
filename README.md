ViGo - Vi in Go
===============

[![Build Status](https://drone.io/github.com/kisielk/vigo/status.png)](https://drone.io/github.com/kisielk/vigo/latest)

ViGo is an attempt to implement a vim-like text editor in Go, while maintaining a lean feature set. 
The project is still in its early days and definitely not suitable for daily use yet.

It's based on nsf's religious [godit](https://github.com/nsf/godit)

Contributing
------------

ViGo is still in its early days, though contributions are welcome. In order to get started, you will first need to fork the respository.

Due to the way that golang imports work, the way you work on the project will differ slightly to other Github projects. Instead of cloning your fork locally, you will instead need to clone the original repository in `$GOPATH/src/github.com/kisielk/vigo/`. It is important that it be at this path, and all the imports in the application will look for packages here. You can clone the project using git, or simply run `go get github.com/kisielk/vigo/`.

Once you have the repository at the above location, you will need to add your fork as a git remote. You can do this with the following command: `git remote add fork <url of your fork>`. Now, when you work on a feature, you can push to your fork, create a pull request, and then update from origin when your change is merged.

The complete workflow could be:

``` 
$ cd $GOPATH
$ go get github.com/kisielk/vigo/
$ cd src/github.com/kisielk/vigo/
$ git remote add fork <url of your fork>
#* hack hack hack*
$ git push fork
# create github pull request
$ git pull origin master
```

Building
--------

In the project directory, you can build ViGo by running `go build`. This will produce a `vigo` binary which you can run with `./vigo`.

# Oracle plugin for Go-between


This plugin requires Oracle Instant Client files to be available in order to run.

Instructions on how to configure your environment to run the plugin and its
tests are here: https://oracle.github.io/odpi/doc/installation.html

Because setting the build server up to build and test this will be a pain, 
thanks to Oracle, it's not done yet. You'll need to build, test and deploy
any changes yourself.

## Build using docker
```
docker build --output type=local,dest=.\build\docker -f Dockerfile .
```

## Build and deploy

To build for a platform, run `mage buildLinux` or `mage buildWindows` (see note below about building Windows).

To build for both platforms, run `mage build`.

To build and upload to a go-between hub, run 

```
UPLOAD=env mage -v build
```

Where `env` can be "red", "green", or "blue".  

> You must have VAULT_ADDR and VAULT_TOKEN set for the environment you're targeting.



### Notes about building:

To build the Windows binary on Linux you need to install the right compiler:
```
sudo apt-get install mingw-w64 -y
```

Then you can build using this command:
 ```
CGO_ENABLED=1 GOOS=windows CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ go build
```

It's really slow.
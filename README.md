# Oracle plugin for Go-between


This plugin requires Oracle Instant Client files to be available in order to run.

Instructions on how to configure your environment to run the plugin and its
tests are here: https://oracle.github.io/odpi/doc/installation.html





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
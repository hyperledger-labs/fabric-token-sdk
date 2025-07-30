# Generation Mock Files

We use `counterfeiter` to generate mocks. 

For example, `token/driver/mock` contains mock auto-generated files for the Driver API..
The generation directives can be found close to the interface that needs to be mocked, for example at `token/driver/transfer.go`:
```
//go:generate counterfeiter -o mock/ts.go -fake-name TransferService . TransferService
```
To regenerate all mock files, go to `token/driver` and run `go generate`.
The same applies to other packages containing the `//go:generate` directive. 

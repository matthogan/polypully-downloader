# Developer

## Prereqs

To enable compilation, install the following package for the test files:-

```shell
$ go get github.com/golang/mock/gomock
```

## Vscode debugging

See [.vscode/launch.json](.vscode/launch.json) for debug configurations.

## Tools

May need to install delve, vscode may
do this automatically with the [`Go`](https://marketplace.visualstudio.com/items?itemName=golang.go) plugin.

```shell
$ brew install delve
```

Goreleaser

```shell
$ brew install goreleaser
```

(and `make` if using the script.)

## Generated help docs

```bash
$ cmd/help/gendocs.sh
```

## Sonar

Deploy test results and coverage stats to a sonar configuration.

See the [Makefile](Makefile) for options.

```shell
$ make sonar
```

Options

```make
SONAR_URL ?= http://localhost:9000
SONAR_KEY ?= github.com.matthogan.zc.cmd.cn
SONAR_TOKEN ?= 36bf40c0d7cd898009d4bf5d2b52483c0743f025
```

## Mocking

The [gomock](https://github.com/golang/mock) package generates mocks of any `interface` type that it
discovers in a file that it has been told to introspect.

Install the following to generate mocks:-

```shell
$ go install github.com/golang/mock/mockgen@v1.6.0
```

This installs a tool called `mockgen` in the `$(go env GOPATH)/bin` directory. A
make target can be invoked to generate mocks into the [mock](mock/) directory:-

```shell
$ make mockgen SOURCE=pkg/forge/fetch.go PACKAGE=forge DEST=pkg/forge/fetch_mock.go
```

Those mocks are generated from a file in the project, however mocks may
also be generated from downloaded packages that are in the `GOPATH` or vendor directory:-

```shell
$ make mockgen SOURCE=pkg/resources/resources.go PACKAGE=resources DEST=pkg/container/resources_mock.go
```

See the mockgen target in the [Makefile](Makefile) for how to run the command standalone.

## Unit testing

The unit tests are a mix of plain func calls, monkey patching, and mocking.

Practical unit testing for even remotely complex code, i.e. with simple dependencies, apparently 
requires a particular code pattern where interfaces, structs and funcs are
defined at a file and/or package scope. The interfaces expose a public api and enable mocks to be
used in the place of the public funcs from where they are being invoked.

The following is a func in a file in a package that has been exposed with an interface.

```go
package ann
// scope
type Ann struct{}
// public api
type AnnApi interface {
	GetAnnThingy(name string) (string, error)
}
// impl
func (a *Ann) GetAnnThingy(name string) (string, error) {
    ...
}
```

The `mockgen` tool should be used to generate mocks based on the `AnnApi` interface:-

```shell
$ make mockgen SOURCE=pkg/ann/ann.go PACKAGE=ann DEST=pkg/ann/ann_mock.go
```

If a func in some other package wants to call `GetAnnThingy` it will define the `Ann` struct and invoke it.

```go
package bnn

import a "pkg/ann"

var ann a.AnnApi // this gets monkey patched

func init() { // special
    ann = a.Ann{}
}

type Bnn struct {}

func (b *Bnn) DoAnnThingy() {
    s, err := ann.GetAnnThingy("abcd")
    ...
}
```

A unit test for `DoAnnThingy` defined in a different test file can use the generated mocks to get
the `ann.GetAnnThingy` func to return whatever it likes by abusing the expectations pattern for instance:-

```go
package bnn

import (
    "testing"
    // real package
    a "github.com/foo/pkg/ann"
    // importing the generated mocks
    mock "github.com/foo/mock/pkg/ann"
    "github.com/golang/mock/gomock"
)

func TestDoAnnThingy(t *testing.T) {
    controller := gomock.NewController(t)
    // monkey patch the ann var
    ann = mock.NewMockAnnApi(controller)
    // return "defg" when the inner dependency is invoked
    ann.(*mock.NewMockAnnApi).EXPECT().GetAnnThingy(gomock.Any()).
        Return("defg", nil)
    b := Bnn{}
    actual, err := b.DoAnnThingy("abcd")
    ...
}
```

Unit tests are in the same directories as the main source so that the coverage tool will work. A
dedicated [test](test) directory would be a more natural home for test code but that is not
an option now. Test code is not compiled into the main binary unless there's an undetected rogue 
import in the main code or forgotten init func in the test code for instance.

## Huge binary

The [goweight](https://github.com/jondot/goweight)
tool can be used to inspect a go binary and work out what
packages are taking up the most space.

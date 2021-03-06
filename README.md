# nserve - server startup/shutdown in an nject world

[![GoDoc](https://godoc.org/github.com/muir/nserve?status.png)](https://pkg.go.dev/github.com/muir/nserve)
![unit tests](https://github.com/muir/nserve/actions/workflows/go.yml/badge.svg)
[![report card](https://goreportcard.com/badge/github.com/muir/nserve)](https://goreportcard.com/report/github.com/muir/nserve)
[![codecov](https://codecov.io/gh/muir/nserve/branch/main/graph/badge.svg)](https://codecov.io/gh/muir/nserve)

Install:

	go get github.com/muir/nserve

---

Prior to [nject](https://github.com/muir/nject) version 0.2.0, this was part of that repo.

---

This package provides server startup and shutdown wrappers that can be used
with libraries and servers.  It is based up [nject](https://github.com/muir/nject).

### Example

On thing you might want to do with nserve is to use a `Hook` to trigger
per-library database migrations using [libschema](https://github.com/muir/libschema).

First create the hook:

```go
package myhooks

import "github.com/nject/nserve"

var MigrateMyDB = nserve.NewHook("migrate, nserve.Ascending)
```

In each library, have a create function:

```go
package users

import(
	"github.com/muir/libschema/lspostgres"
	"github.com/muir/nject/nserve"
)

func NewUsersStore(app *nserve.App) *Store {
	...
	app.On(myhooks.MigrateMyDB, func(database *libschema.Database) {
		database.Migrations("MyLibrary",
			lspostgres.Script("create users", `
				CREATE TABLE users (
					id	bigint PRIMARY KEY,
					name	text
				)
			`),
		)
	})
	...
	return &Store{}
}
```

Then as part of server startup, invoke the migration hook:

```go
package main

import(
	"github.com/muir/libschema"
	"github.com/muir/libschema/lspostgres"
	"github.com/muir/nject"
)

func main() {
	app, err := nserve.CreateApp("myApp", users.NewUserStore, ...)
	schema := libschema.NewSchema(ctx, libschema.Options{})
	sqlDB, err := sql.Open("postgres", "....")
	database, err := lspostgres.New(logger, "main-db", schema, sqlDB)
	myhooks.MigrateMyDB.Using(database)
	err = app.Do(myhooks.MigrateMyDB)
```


### How to structure your application

Libraries become injected dependencies.  They can in turn have other libraries
as dependencies of them.  Since only the depenencies that are required are 
actaully injected, the easiest thing is to have a master list of all your libraries
and provide that to all your apps.

Let's call that master list `allLibraries`.

```go
app, err := nserve.CreateApp("myApp", allLibrariesSequence, createAppFunction)
err = app.Do(nserve.Start)
err = app.Do(nserve.Stop)
```

### Hooks

Libaries and appliations can register callbacks on a per-hook basis.  Two hooks
are pre-provided by other hooks can be created.

Hook invocation can be limited by a timeout.  If the hook does not complete in
that amount of time, the hook will return error and continue processing in the
background.

The Start, Stop, and Shutdown hooks are pre-defined:

```go
var Shutdown = NewHook("shutdown", ReverseOrder)
var Stop = NewHook("stop", ReverseOrder).OnError(Shutdown).ContinuePastError(true)
var Start = NewHook("start", ForwardOrder).OnError(Stop)
```

Hooks can be invoked in registration order (`ForwardOrder`) or in 
reverse registration order `ReverseOrder`.  

If there is an `OnError` modifier for the hook registration, then that
hook gets invoked if there is an error returned when the hook is running.

Libraries and applications can register callbacks for hooks by taking an
`*nserve.App` as as an input parameter and then using that to register callbacks:

```go
func CreateMyLibrary(app *nserve.App) *MyLibrary {
	lib := &MyLibrary{ ... }
	app.On(Start, lib.Start)
	return lib
}
func (lib *MyLibrary) Start(app *nserve.App) {
	app.On(Stop, lib.Stop)
}
```

The callback function can be any nject injection chain.  If it ends with a
function that can return
error, then any such error will be become the error return from `app.Do` and if
there is an `OnError` handler for that hook, that handler will be invoked.


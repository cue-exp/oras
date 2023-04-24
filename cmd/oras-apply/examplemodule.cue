package test

modules: "example.com@v0.0.1": {
	deps: [
		modules["foo.com/bar/hello@v0.2.3"],
		modules["bar.com@v0.5.0"],
	]
	files: """
		-- cue.mod/module.cue --
		module: "example.com"
		deps: {
			"foo.com/bar/hello": v: "v0.2.3",
			"bar.com": v: "v0.5.0",
		}
		-- top.cue --
		package main

		import a "foo.com/bar/hello"
		a
		main: "main"
		"example.com": "v0.0.1"
		"""
}
modules: "foo.com/bar/hello@v0.2.3": {
	deps: [
		modules["bar.com@v0.0.2"],
		modules["baz.org@v0.10.1"],
	]
	files: """
		-- cue.mod/module.cue --
		module: "foo.com/bar/hello"
		deps: {
			"bar.com": v: "v0.0.2"
			"baz.org": v: "v0.10.1"
		}
		-- x.cue --
		package hello
		import (
			a "bar.com/bar"
			b "baz.org:baz"
		)
		"foo.com/bar/hello": "v0.2.3"
		a
		b
		"""
}
modules: "bar.com@v0.0.2": {
	deps: [
		modules["baz.org@v0.0.2"],
	]
	files: """
		-- cue.mod/module.cue --
		module: "bar.com"
		deps: "baz.org": v: "v0.0.2"
		-- x.cue --
		package bar
		import a "baz.org:baz"
		"bar.com": "v0.0.2"
		a
		"""
}
modules: "bar.com@v0.5.0": {
	deps: [
		modules["baz.org@v0.5.0"],
	]
	files: """
		-- cue.mod/module.cue --
		module: "bar.com"
		deps: "baz.org": v: "v0.5.0"
		-- x.cue --
		package bar
		import a "baz.org:baz"
		"bar.com": "v0.0.2"
		a
		"""
}
modules: "baz.org@v0.0.2": {
	deps: []
	files: """
		-- cue.mod/module.cue --
		module: "baz.org"
		-- baz.cue --
		package baz
		"baz.org": "v0.0.2"
		"""
}
modules: "baz.org@v0.1.2": {
	deps: []
	files: """
		-- cue.mod/module.cue --
		module: "baz.org"
		-- x.cue --
		package baz
		"baz.org": "v0.1.2"
		"""
}
modules: "baz.org@v0.5.0": {
	deps: []
	files: """
		-- cue.mod/module.cue --
		module: "baz.org"
		-- baz.cue --
		package baz
		"baz.org": "v0.5.0"
		"""
}
modules: "baz.org@v0.10.1": {
	deps: []
	files: """
		-- cue.mod/module.cue --
		module: "baz.org"
		-- baz.cue --
		package baz
		"baz.org": "v0.10.1"
		"""
}

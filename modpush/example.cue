package modpush

modules: "example.com@v0.0.1": {
	deps: [
		modules["foo.com/bar/hello@v0.2.3"],
		modules["bar.com@v0.5.0"],
	]
	moduleFile: {
		module: "example.com"
		deps: {
			"foo.com/bar/hello": v: "v0.2.3"
			"bar.com": v:           "v0.5.0"
		}
	}
	files: "top.cue": """
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
	moduleFile: {
		module: "foo.com/bar/hello"
		deps: {
			"bar.com": v: "v0.0.2"
			"baz.org": v: "v0.10.1"
		}
	}
	files: "x.cue": """
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
	moduleFile: {
		module: "bar.com"
		deps: "baz.org": v: "v0.0.2"
	}
	files: "x.cue": """
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
	moduleFile: {
		module: "bar.com"
		deps: "baz.org": v: "v0.5.0"
	}
	files: "x.cue": """
		package bar
		import a "baz.org:baz"
		"bar.com": "v0.0.2"
		a
		"""
}
modules: "baz.org@v0.0.2": {
	deps: []
	moduleFile: module: "baz.org"
	files: "baz.cue": """
		package baz
		"baz.org": "v0.0.2"
		"""
}
modules: "baz.org@v0.1.2": {
	deps: []
	moduleFile: module: "baz.org"
	files: "x.cue": """
		package baz
		"baz.org": "v0.1.2"
		"""
}
modules: "baz.org@v0.5.0": {
	deps: []
	moduleFile: module: "baz.org"
	files: "baz.cue": """
		package baz
		"baz.org": "v0.5.0"
		"""
}
modules: "baz.org@v0.10.1": {
	deps: []
	moduleFile: module: "baz.org"
	files: "baz.cue": """
		package baz
		"baz.org": "v0.10.1"
		"""
}

package main

func init() {
	excludeDirs["plugin"] = true // https://github.com/jcburley/joker/issues/19: causes intermittent hangs on OS X merely by being present
}

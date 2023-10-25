After the default ol worker started. use **./ol pprof cpu-start** to start the pprof.
Use **./ol pprof cpu-stop** to stop the pprof. There will be a cpu.prof file. use 
`go tool pprof -http=localhost:8889 cpu.prof` and go to locoalhost:8889 to see the result.

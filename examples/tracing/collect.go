package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"

	"github.com/pyroscope-io/dotnetdiag"
	"github.com/pyroscope-io/dotnetdiag/nettrace"
	"github.com/pyroscope-io/dotnetdiag/nettrace/profiler"
)

func main() {
	var ps string
	flag.StringVar(&ps, "p", "", "Target process ID")
	flag.Parse()

	pid, err := strconv.Atoi(ps)
	if err != nil {
		log.Fatalln("Invalid PID:", err)
	}

	c := dotnetdiag.NewClient(dotnetdiag.DefaultServerAddress(pid))
	ctc := dotnetdiag.CollectTracingConfig{
		CircularBufferSizeMB: 10,
		Providers: []dotnetdiag.ProviderConfig{
			{
				Keywords:     0x0000F00000000000,
				LogLevel:     4,
				ProviderName: "Microsoft-DotNETCore-SampleProfiler",
			},
		},
	}

	sess, err := c.CollectTracing(ctc)
	if err != nil {
		log.Fatalln(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		log.Println("Interrupted")
		if err = sess.Close(); err != nil {
			log.Fatalln(err)
		}
	}()

	// Process the stream with the sample profiler.
	stream := nettrace.NewStream(sess)
	trace, err := stream.Open()
	if err != nil {
		_ = sess.Close()
		log.Fatalln(err)
	}

	p := profiler.NewSampleProfiler(trace)
	stream.EventHandler = p.EventHandler
	stream.MetadataHandler = p.MetadataHandler
	stream.StackBlockHandler = p.StackBlockHandler
	stream.SequencePointBlockHandler = p.SequencePointBlockHandler

	log.Println("Collecting trace log")
	for {
		switch err = stream.Next(); err {
		default:
			log.Fatalln(err)
		case nil:
			continue
		case io.EOF:
			for name, time := range p.Samples() {
				log.Println(name, time)
			}
			return
		}
	}
}

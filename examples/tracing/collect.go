package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/pyroscope-io/dotnetdiag"
)

func main() {
	var (
		socketFilePath string
		outputFilePath = "my-traces.nettrace"
	)

	flag.StringVar(&socketFilePath, "s", "", "Path to Diagnostic IPC socket")
	flag.StringVar(&outputFilePath, "o", "my-traces.nettrace", "Output file.")
	flag.Parse()

	if socketFilePath == "" {
		log.Fatalln("Diagnostic IPC socket path is required.")
	}

	file, err := os.Create(outputFilePath)
	if err != nil {
		log.Fatalln(err)
	}
	defer file.Close()

	c := dotnetdiag.NewClient(socketFilePath)
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

	if _, err = io.Copy(file, sess); err != nil {
		log.Fatalln(err)
	}

	log.Println("Done")
}

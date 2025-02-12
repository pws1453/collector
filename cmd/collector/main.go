package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"collector/pkg/archive"
	"collector/pkg/cli"
	"collector/pkg/logger"
	"collector/pkg/req"

	"golang.org/x/sync/errgroup"
)

// Version comes from CI
var (
	version string
	args    Args
)

func pause(msg string) {
	fmt.Println("Press enter to exit.")
	var throwaway string
	fmt.Scanln(&throwaway)
}

func main() {
	log, err := logger.New("collector.log")
	if err != nil {
		panic(err)
	}
	args = newArgs()
	cfg := cli.Config{
		Host:              args.APIC,
		Username:          args.Username,
		Password:          args.Password,
		RetryDelay:        args.RetryDelay,
		RequestRetryCount: args.RequestRetryCount,
		BatchSize:         args.BatchSize,
	}

	// Initialize ACI HTTP client
	client, err := cli.GetClient(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Error initializing ACI client.")
	}

	// Create results archive
	arc, err := archive.NewWriter(args.Output)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error creating archive file: %s.", args.Output)
	}
	defer arc.Close()

	// Initiate requests
	reqs, err := req.GetRequests()
	if err != nil {
		log.Fatal().Err(err).Msgf("Error reading requests.")
	}

	// Batch and fetch queries in parallel
	batch := 1
	for i := 0; i < len(reqs); i += args.BatchSize {
		var g errgroup.Group
		fmt.Println(strings.Repeat("=", 30))
		fmt.Println("Fetching request batch", batch)
		fmt.Println(strings.Repeat("=", 30))
		for j := i; j < i+args.BatchSize && j < len(reqs); j++ {
			req := reqs[j]
			g.Go(func() error {
				return cli.FetchResource(client, req, arc, cfg)
			})
		}
		err = g.Wait()
		if err != nil {
			log.Error().Err(err).Msg("Error fetching data.")
		}
		batch++
	}

	fmt.Println(strings.Repeat("=", 30))
	fmt.Println("Complete")
	fmt.Println(strings.Repeat("=", 30))

	path, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot read current working directory")
	}
	outPath := filepath.Join(path, args.Output)

	if err != nil {
		log.Warn().Err(err).Msg("some data could not be fetched")
		log.Info().Err(err).Msgf("Available data written to %s.", outPath)
	} else {
		log.Info().Msg("Collection complete.")
		log.Info().Msgf("Please provide %s to Cisco Services for further analysis.", outPath)
	}
	pause("Press enter to exit.")
}

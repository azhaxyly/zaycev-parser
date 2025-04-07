package config

import "flag"

type Config struct {
	Limit    int
	Output   string
	Download bool
	Period   string
}

func ParseFlags() Config {
	limit := flag.Int("limit", 50, "How many tracks to download")
	output := flag.String("output", "json", "Output format (json or csv)")
	download := flag.Bool("download", false, "Download tracks or not")
	period := flag.String("period", "day", "Period for fetching tracks (day, week, month)")

	flag.Parse()

	return Config{
		Limit:    *limit,
		Output:   *output,
		Download: *download,
		Period:   *period,
	}
}

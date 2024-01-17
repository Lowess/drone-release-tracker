package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"

	"github.com/nikolaydubina/calendarheatmap/charts"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var (
	fromFlag   = flag.String("from", getQuarterStartDate(time.Now()).Format("2006-01-02"), "Releases after this date will be included")
	toFlag     = flag.String("to", getQuarterEndDate(time.Now()).Format("2006-01-02"), "Releases before this date will be included")
	reposFlag  = flag.String("repos", "octocat/demo", "Name of the repository to list releases for")
	outputFlag = flag.String("output", "png", "Output format (json, png, jpeg, gif, svg)")
)

var (
	token = os.Getenv("DRONE_TOKEN")
	host  = os.Getenv("DRONE_SERVER")
)

//go:embed assets/fonts/Sunflower-Medium.ttf
var defaultFontFaceBytes []byte

//go:embed assets/colorscales/green-blue-9.csv
var defaultColorScaleBytes []byte

// Return a list of builds that started after a given date.
func filterReleasesByDate(builds []*drone.Build, from time.Time, to time.Time) []*drone.Build {
	filteredBuilds := make([]*drone.Build, 0)

	for _, build := range builds {
		// Only Include releases (eg. promoted builds
		if build.Event == "promote" && build.Deploy == "production" {
			buildTime := time.Unix(build.Created, 0)
			if buildTime.After(from) && buildTime.Before(to) {
				// Include the build if its creation date is after the given date.
				filteredBuilds = append(filteredBuilds, build)
			}
		}
	}

	return filteredBuilds
}

// Find all releases until a given date
func findAllReleasesWithinRange(client drone.Client, namespace string, repo string, from time.Time, to time.Time, page int) []*drone.Build {

	paginatedBuilds, err := client.BuildList(namespace, repo, drone.ListOptions{Page: page, Size: 25})
	if err != nil || len(paginatedBuilds) == 0 {
		return nil
	}

	// Recursion end condition: First build item fetched from paginated response is older than until date
	oldestBuild := paginatedBuilds[0]
	oldestBuildTime := time.Unix(oldestBuild.Created, 0)

	if oldestBuildTime.Before(from) {
		return nil
	}

	filteredReleases := filterReleasesByDate(paginatedBuilds, from, to)
	nextReleases := findAllReleasesWithinRange(client, namespace, repo, from, to, page+1)
	return append(filteredReleases, nextReleases...)
}

func getQuarterStartDate(currentDate time.Time) time.Time {
	year, month, _ := currentDate.Date()
	quarterStartMonth := time.Month(((int(month)-1)/3)*3 + 1)
	return time.Date(year, quarterStartMonth, 1, 0, 0, 0, 0, currentDate.Location())
}

func getQuarterEndDate(currentDate time.Time) time.Time {
	year, month, _ := currentDate.Date()
	quarterStartMonth := time.Month(((int(month)-1)/3)*3 + 1)
	quarterEndMonth := quarterStartMonth + 2 // The end month is three months after the start month.
	quarterEndMonth %= 12                    // Ensure the month doesn't exceed 12 (December).
	if quarterEndMonth < quarterStartMonth {
		year++
	}
	lastDayOfQuarter := time.Date(year, quarterEndMonth+1, 0, 0, 0, 0, 0, currentDate.Location())
	return lastDayOfQuarter
}

// Taken from https://github.com/nikolaydubina/calendarheatmap/blob/master/main.go.
func plotHeatmap(data []byte, outputFormat string) {

	labels := true
	monthSep := true
	colorScale := "green-blue-9.csv"
	locale := "en_US"

	var colorscale charts.BasicColorScale
	if assetsPath := os.Getenv("CALENDAR_HEATMAP_ASSETS_PATH"); assetsPath != "" {
		var err error
		colorscale, err = charts.NewBasicColorscaleFromCSVFile(path.Join(assetsPath, "colorscales", colorScale))
		if err != nil {
			log.Fatal(err)
		}
	} else {
		var err error
		if colorScale != "green-blue-9.csv" {
			log.Printf("defaulting to colorscale %s since CALENDAR_HEATMAP_ASSETS_PATH is not set", "green-blue-9.csv")
		}
		colorscale, err = charts.NewBasicColorscaleFromCSV(bytes.NewBuffer(defaultColorScaleBytes))
		if err != nil {
			log.Fatal(err)
		}
	}

	fontFace, err := charts.LoadFontFace(defaultFontFaceBytes, opentype.FaceOptions{
		Size:    26,
		DPI:     280,
		Hinting: font.HintingNone,
	})
	if err != nil {
		log.Fatal(err)
	}

	var counts map[string]int
	if err := json.Unmarshal(data, &counts); err != nil {
		log.Fatal(err)
	}
	conf := charts.HeatmapConfig{
		Counts:              counts,
		ColorScale:          colorscale,
		DrawMonthSeparator:  monthSep,
		DrawLabels:          labels,
		Margin:              30,
		BoxSize:             150,
		MonthSeparatorWidth: 5,
		MonthLabelYOffset:   50,
		TextWidthLeft:       300,
		TextHeightTop:       200,
		TextColor:           color.RGBA{100, 100, 100, 255},
		BorderColor:         color.RGBA{200, 200, 200, 255},
		Locale:              locale,
		Format:              outputFormat,
		FontFace:            fontFace,
		ShowWeekdays: map[time.Weekday]bool{
			time.Monday:    true,
			time.Wednesday: true,
			time.Friday:    true,
		},
	}
	charts.WriteHeatmap(conf, os.Stdout)
}

type DroneRepo string

func (r DroneRepo) Split(str string) (string, string, error) {
	s := strings.Split(string(r), str)
	if len(s) < 2 {
		return "", "", errors.New("Drone repository should be made of <namespace>/<name>")
	}
	return s[0], s[1], nil
}

func main() {
	// Parse flags
	flag.Parse()

	// create an http client with oauth authentication.
	config := new(oauth2.Config)
	author := config.Client(
		context.Background(),
		&oauth2.Token{
			AccessToken: token,
		},
	)

	// create the drone client with authenticator
	client := drone.NewClient(host, author)

	// find all releases within date range
	fromDate, _ := time.Parse("2006-01-02", *fromFlag)
	toDate, _ := time.Parse("2006-01-02", *toFlag)

	droneRepos := DroneRepo(*reposFlag)

	var releases []*drone.Build

	// Loop through a list of CSV repos and find all releases within date range
	for _, droneRepo := range strings.Split(string(droneRepos), ",") {
		namespace, name, err := DroneRepo(droneRepo).Split("/")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		// Extend releases array with releases from this repo
		releases = append(releases, findAllReleasesWithinRange(client, namespace, name, fromDate, toDate, 1)...)
	}

	heatmapData := make(map[string]int)
	for _, build := range releases {
		releaseDate := time.Unix(build.Created, 0)
		heatmapData[releaseDate.Format("2006-01-02")] += 1
	}

	heatmapDataJson, _ := json.MarshalIndent(heatmapData, "", "")

	if *outputFlag == "json" {
		fmt.Println(string(heatmapDataJson))
	} else {
		// Plot heatmap
		plotHeatmap(heatmapDataJson, *outputFlag)
	}

}

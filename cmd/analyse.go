package cmd

import (
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/giantswarm/cloud-carbon/pkg/footprint"
	"github.com/olekukonko/tablewriter"

	"github.com/spf13/cobra"
)

var analyseCmd = &cobra.Command{
	Use:   "analyse PATH",
	Short: "Analyse an AWS usage report",
	Long: `Analyse an AWS usage report.

The input file, specified by PATH, must be a gzipped CSV file in the format
"hourly usage without IDs".

As a result, the EC2 usage by region and instance will be printed.
`,
	Run:  analyse,
	Args: cobra.MinimumNArgs(1),
}

const (
	headerBillingPeriodEndDate   = "bill/BillingPeriodEndDate"
	headerBillingPeriodStartDate = "bill/BillingPeriodStartDate"
	headerBillPayerAccountID     = "bill/PayerAccountId"
	headerIdentityTimeInterval   = "identity/TimeInterval"
	headerLineItemLineItemType   = "lineItem/LineItemType"
	headerLineItemOperation      = "lineItem/Operation"
	headerLineItemProductCode    = "lineItem/ProductCode"
	headerLineItemUsageAccountID = "lineItem/UsageAccountId"
	headerLineItemUsageEndDate   = "lineItem/UsageEndDate"
	headerLineItemUsageStartDate = "lineItem/UsageStartDate"
	headerProductInstanceType    = "product/instanceType"
	headerProductProductFamily   = "product/productFamily"
	headerProductRegionCode      = "product/regionCode"

	dateTimeLayout = "2006-01-02T15:04:05Z"
)

var (
	headers map[string]int
)

type ReportRow struct {
	PayerAccountID string
	UsageAccountID string
	Region         string
	InstanceType   string
	UsageStartTime time.Time
	UsageEndTime   time.Time
	Duration       time.Duration
}

type AggregateReportRow struct {
	Region        string
	InstanceType  string
	Duration      time.Duration
	EmissionGrams float64
}

func readReportRow(fields []string) ReportRow {
	r := ReportRow{
		PayerAccountID: fields[headers[headerBillPayerAccountID]],
		UsageAccountID: fields[headers[headerLineItemUsageAccountID]],
		Region:         fields[headers[headerProductRegionCode]],
		InstanceType:   fields[headers[headerProductInstanceType]],
		UsageStartTime: mustParseDate(fields[headers[headerLineItemUsageStartDate]]),
		UsageEndTime:   mustParseDate(fields[headers[headerLineItemUsageEndDate]]),
	}

	// Fancy logic to basically compute a duration of one hour.
	interval := fields[headers[headerIdentityTimeInterval]]
	parts := strings.Split(interval, "/")
	r.UsageStartTime = mustParseDate(parts[0])
	r.UsageEndTime = mustParseDate(parts[1])
	r.Duration = r.UsageEndTime.Sub(r.UsageStartTime)

	return r
}

func mustParseDate(s string) time.Time {
	dateTime, _ := time.Parse(dateTimeLayout, s)
	return dateTime
}

func formatGrams(g float64) string {
	if g > (1000 * 1000) {
		return fmt.Sprintf("%.1f MTCO2e", g/1000/1000)
	}
	if g > 1000 {
		return fmt.Sprintf("%.1f kgCO2e", g/1000)
	}
	return fmt.Sprintf("%.0f gCO2e", g)
}

func analyse(cmd *cobra.Command, args []string) {
	path := args[0]
	fmt.Printf("Analysing report from path %s\n", path)

	gzFile, err := os.Open(path)
	if err != nil {
		log.Fatalf("Could not open file: %s", err)
	}
	defer gzFile.Close()

	csvFile, err := gzip.NewReader(gzFile)
	if err != nil {
		log.Fatalf("Could not uncompress file: %s", err)
	}
	defer csvFile.Close()

	processedHeaders := false
	lineCount := 0
	headers = make(map[string]int)
	earliestDate := mustParseDate("2100-12-31T23:59:59Z")
	latestDate := mustParseDate("0000-00-00T00:00:00Z")

	// Aggregate report rows where key is in the form of
	// region_instancetype
	aggregate := make(map[string]AggregateReportRow)

	fcsv := csv.NewReader(csvFile)
	for {
		csvRecord, err := fcsv.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("ERROR: ", err.Error())
			break
		}

		if !processedHeaders {
			for index, field := range csvRecord {
				headers[field] = index
			}
			processedHeaders = true
		}

		// Filtering out everything that is not EC2 instance usage
		if csvRecord[headers[headerLineItemLineItemType]] != "Usage" {
			continue
		}
		if csvRecord[headers[headerLineItemProductCode]] != "AmazonEC2" {
			continue
		}
		if csvRecord[headers[headerProductProductFamily]] != "Compute Instance" {
			continue
		}
		if !strings.HasPrefix(csvRecord[headers[headerLineItemOperation]], "RunInstances") {
			continue
		}

		lineCount++

		r := readReportRow(csvRecord)
		key := fmt.Sprintf("%s_%s", r.Region, r.InstanceType)
		val, exists := aggregate[key]
		if exists {
			val.Duration += r.Duration
			aggregate[key] = val
		} else {
			aggregate[key] = AggregateReportRow{
				Region:       r.Region,
				InstanceType: r.InstanceType,
				Duration:     r.Duration,
			}
		}

		if r.UsageStartTime.Before(earliestDate) {
			earliestDate = r.UsageStartTime
		}
		if r.UsageEndTime.After(latestDate) {
			latestDate = r.UsageEndTime
		}
	}

	fmt.Printf("Processed %d lines about EC2 usage.\n", lineCount)
	fmt.Printf("Time range covered: %s - %s (%s).\n\n", earliestDate, latestDate, latestDate.Sub(earliestDate))

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Region", "Instance type", "Duration", "Emissions"})

	var aggregateReportRows []AggregateReportRow
	var total float64

	for key := range aggregate {
		result, err := footprint.AWS(aggregate[key].Region, aggregate[key].InstanceType, aggregate[key].Duration)
		if err != nil {
			log.Printf("Error for key %s: %s", key, err)
			continue
		}

		aggregateReportRows = append(aggregateReportRows, AggregateReportRow{
			Region:        aggregate[key].Region,
			InstanceType:  aggregate[key].InstanceType,
			Duration:      aggregate[key].Duration,
			EmissionGrams: result,
		})

		total += result
	}

	sort.Slice(aggregateReportRows, func(i, j int) bool {
		return aggregateReportRows[i].InstanceType < aggregateReportRows[j].InstanceType
	})
	sort.Slice(aggregateReportRows, func(i, j int) bool {
		return aggregateReportRows[i].Region < aggregateReportRows[j].Region
	})

	for _, row := range aggregateReportRows {
		table.Append([]string{
			row.Region,
			row.InstanceType,
			row.Duration.String(),
			formatGrams(row.EmissionGrams),
		})
	}

	table.SetFooter([]string{"", "", "Total", formatGrams(total)})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetFooterAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)
	table.SetColumnSeparator("")
	table.SetCenterSeparator("")
	table.SetRowSeparator("")
	table.SetBorder(false)
	table.SetTablePadding("   ")
	table.Render()
}

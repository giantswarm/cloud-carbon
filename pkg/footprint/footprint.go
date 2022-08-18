// Package footprint provides data and functions
// to estimate the carbon emissions of AWS EC2
// instance operation.
//
// Data source: https://docs.google.com/spreadsheets/d/1DqYgQnEDLQVQm5acMAhLgHLD8xXCG9BIrk-_Nv6jF3k/edit#gid=504755275
// Data and methodology provided by Teads engineering, under the
// Creative Commons Attribution 4.0 International License.
//
// Data snapshot date: 2022-08-17
//
// More background on the methodology:
// https://medium.com/teads-engineering/building-an-aws-ec2-carbon-emissions-dataset-3f0fd76c98ac
package footprint

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
)

//go:embed aws-ec2-instances.csv
var ec2instancesCSV string

//go:embed aws-regions.csv
var awsRegionsCSV string

// ec2instances stores data about EC2 instances, using the instance type name as key.
var ec2instances map[string]EC2Instance

// awsRegions stores data about AWS regions, using the region code as key.
var awsRegions map[string]AWSRegion

type EC2Instance struct {
	// WattAt50Percent is the instance power consumtion in Watt at 50% load
	PowerAt50Percent float64

	// ManufacturingEmissionsHourly is the emissions created during production of the
	// hardware, calculated as contribution to the hourly footprint, in metric grams CO2e.
	ManufacturingEmissionsHourly float64
}

type AWSRegion struct {
	// CarbonIntensity is the amount of CO2 emitted when producing electricity.
	// Unit: metric gram per kilowatt hour.
	CarbonIntensity float64

	// PUE is the power usage effectiveness coefficient of the data center.
	// See https://en.wikipedia.org/wiki/Power_usage_effectiveness for details.
	PUE float64
}

func init() {
	err := readEC2Instances()
	if err != nil {
		log.Fatal(err)
	}

	err = readAWSRegions()
	if err != nil {
		log.Fatal(err)
	}
}

func readEC2Instances() error {
	reader := csv.NewReader(strings.NewReader(ec2instancesCSV))
	lineCount := 0
	ec2instances = make(map[string]EC2Instance)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip first row containing column headers.
		lineCount++
		if lineCount == 1 {
			continue
		}

		// Process record.
		// We expect the first column to contain the instance type,
		// 30th column to contain power at 50% load,
		// 37th column to contain manufacturing emissions.
		power, err := strconv.ParseFloat(record[29], 64)
		if err != nil {
			return fmt.Errorf("error parsing %q as float: %s", record[29], err)
		}

		manuf, err := strconv.ParseFloat(record[36], 64)
		if err != nil {
			return fmt.Errorf("error parsing %q as float: %s", record[36], err)
		}

		ec2instances[record[0]] = EC2Instance{
			PowerAt50Percent:             power,
			ManufacturingEmissionsHourly: manuf,
		}
	}

	return nil
}

func readAWSRegions() error {
	reader := csv.NewReader(strings.NewReader(awsRegionsCSV))
	lineCount := 0
	awsRegions = make(map[string]AWSRegion)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Skip first row containing column headers.
		lineCount++
		if lineCount == 1 {
			continue
		}

		// Process record.
		// We expect the first column to contain the region code,
		// 5th column to contain carbon intensity,
		// 7th column to contain PUE.
		carbonIntensity, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			return fmt.Errorf("error parsing carbon intensity %q as float: %s", record[4], err)
		}
		pue, err := strconv.ParseFloat(record[6], 64)
		if err != nil {
			return fmt.Errorf("error parsing PUE %q as float: %s", record[6], err)
		}

		awsRegions[record[0]] = AWSRegion{
			CarbonIntensity: carbonIntensity,
			PUE:             pue,
		}
	}

	return nil
}

// PowerAt50Percent returns the power consumption at 50% load for an EC2 instance type, in watt.
func PowerAt50Percent(ec2InstanceType string) (float64, error) {
	val, exists := ec2instances[ec2InstanceType]
	if !exists {
		return 0, fmt.Errorf("unknown instance type")
	} else {
		return val.PowerAt50Percent, nil
	}
}

// ManufacturingEmissions returns manufacturing emissions for a machine, as an hourly
// contribution to emissions in grams.
func ManufacturingEmissions(ec2InstanceType string) (float64, error) {
	val, exists := ec2instances[ec2InstanceType]
	if !exists {
		return 0, fmt.Errorf("unknown instance type")
	} else {
		return val.ManufacturingEmissionsHourly, nil
	}
}

// CarbonIntensity returns the carbon intensity for an AWS region.
// The return value is the number of grams of CO2 emitted while producing one
// kilowatt hour of electricity for the data center.
func CarbonIntensity(regionCode string) (float64, error) {
	val, exists := awsRegions[regionCode]
	if !exists {
		return 0, fmt.Errorf("unknown AWS region code")
	} else {
		return val.CarbonIntensity, nil
	}
}

// PUE returns the power usage effectiveness coefficient for an AWS region.
// See https://en.wikipedia.org/wiki/Power_usage_effectiveness for details.
func PUE(regionCode string) (float64, error) {
	val, exists := awsRegions[regionCode]
	if !exists {
		return 0, fmt.Errorf("unknown AWS region code")
	} else {
		return val.PUE, nil
	}
}

// AWS returns the footprint in gram CO2 equivalents
func AWS(regionCode, instanceType string, duration time.Duration) (float64, error) {
	pue, err := PUE(regionCode)
	if err != nil {
		return 0, err
	}

	ci, err := CarbonIntensity(regionCode)
	if err != nil {
		return 0, err
	}

	power, err := PowerAt50Percent(instanceType)
	if err != nil {
		return 0, err
	}

	manufacturing, err := ManufacturingEmissions(instanceType)
	if err != nil {
		return 0, err
	}

	powerKiloWatt := power / 1000.0

	hours := float64(duration.Hours())

	//log.Printf("AWS(%s, %s, %s): pue=%v ci=%v power=%v manufacturing=%v hours=%v ", regionCode, instanceType, duration, pue, ci, power, manufacturing, hours)

	return ((powerKiloWatt * pue * ci) + manufacturing) * hours, nil
}

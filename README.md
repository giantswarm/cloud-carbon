![gs-we-are-hiring-1128x191px-v3](https://user-images.githubusercontent.com/273727/185643314-9ff1e016-35de-4f47-b08e-6aad9777e8ba.svg)

# cloud-carbon

A CLI tool to estimate the carbon emissions produced by
AWS EC2 usage.

## Requirements

This tool needs an [AWS Cost and Usage Report](https://docs.aws.amazon.com/cur/latest/userguide/what-is-cur.html) as input. These reports are delivered automatically into an S3 bucket. Usually they cover usage of (up to) one calendar month.

One such report is required to be accessible, e. g. downloaded to the local hard drive. The file is expected to be a gzip compressed comma-separated value (CSV) file.

If you don't have Cost and Usage Reports configured, please check the [AWS documtation](https://docs.aws.amazon.com/cur/latest/userguide/cur-create.html) regarding setting this up.

## Installation

TODO. Short version: clone the repo and build the binary using `go build`. Alternatively, download binary from release.

## Usage

The CLI is invoked as

```nohighlight
cloud-carbon analyse PATH
```

where `PATH` must be replaced with the path to the actual CSV file (gzip compressed). As a result, something like this will get printed:

```nohighlight
Analysing report from path ./hourly-report-without-ids-00001.csv.gz
Processed 18162 lines about EC2 usage.
Time range covered: 2022-08-01 00:00:00 +0000 UTC - 2022-08-18 04:00:00 +0000 UTC (412h0m0s).

  REGION        INSTANCE TYPE  DURATION   EMISSIONS
  eu-central-1  m4.xlarge      1664h0m0s  18.0 kgCO2e
  eu-central-1  m5.xlarge      5836h0m0s  77.8 kgCO2e
  eu-central-1  t3.large       1236h0m0s  8.3 kgCO2e
  eu-central-1  t3.micro       412h0m0s   2.0 kgCO2e
  eu-central-1  t3.small       11h0m0s    57 gCO2e
  eu-west-1     m5.xlarge      3468h0m0s  43.7 kgCO2e
  eu-west-1     t2.medium      412h0m0s   2.4 kgCO2e
  eu-west-1     t2.micro       1236h0m0s  3.4 kgCO2e
  eu-west-1     t3.small       1343h0m0s  6.6 kgCO2e
  eu-west-2     m5.xlarge      2293h0m0s  22.0 kgCO2e
  eu-west-2     t3.small       251h0m0s   965 gCO2e

                                 TOTAL      185.4 KGCO2E
```

## What you get as a result

The output table gives you an aggregation of all EC2 instance usage per region and instance type.

In the last column you get the estimated emissions, expressed as an amount (in g for grams, kg for kilograms, or MT for metric tons) of CO2 equivalents.

The last row contains the sum total of emissions.

In our example above, we see that the input report covers usage from 1st to 18th of August 2022. We see that instances of several types have been run in three different regions.

In order to be able to interpret the result, please read the blog post linked below under Acknowledhememnts. Here is a summary of things to consider.

- The power consumption of an EC2 instance has basically been narrowed down experimentally and averaged. The actual power depends heavily on load. We assume that the instance has an average CPU load of 50 percent.

- The energy mix and the carbon intensity of the electricity for each AWS region is calculated based on recent yearly averages.

- The footprint of machine production is accounted for, based on some reference data and average hardware lifetimes.

- Networking and it's electricity usage is not accounted for.

## Acknowledgements

This tool is based on a methodology and data provided by Teads Engineering.

Data in the `pkg/footprint` folder has been [published](https://docs.google.com/spreadsheets/d/1DqYgQnEDLQVQm5acMAhLgHLD8xXCG9BIrk-_Nv6jF3k/edit#gid=504755275) by Teads under the [Creative Commons Attribution 4.0 International License](https://creativecommons.org/licenses/by/4.0/).

Teads provides an [interactive web UI](https://engineering.teads.com/sustainability/carbon-footprint-estimator-for-aws-instances/) for creating estimates along the same lines.

Detail information regarding the methodology can be found in a [blog post](https://medium.com/teads-engineering/building-an-aws-ec2-carbon-emissions-dataset-3f0fd76c98ac).

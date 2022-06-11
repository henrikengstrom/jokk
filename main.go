package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/jessevdk/go-flags"
	hd "github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"
)

var (
	Sha1ver          string = "UNDEFINED"
	BuildTime        string = "UNDEFINED"
	HeadTags         string = "UNDEFINED"
	HeadVersionTag   string = "UNDEFINED"
	Dirty            string = "UNDEFINED"
	DescribeVersion  string = "UNDEFINED"
	DescribeAll      string = "UNDEFINED"
	LastCommitDate   string = "UNDEFINED"
	LastCommitAuthor string = "UNDEFINED"
	LastCommitEmail  string = "UNDEFINED"
)

type Args struct {
	Version               bool   `short:"v" long:"version" description:"Display version information and exit"`
	CredentialsConfigFile string `short:"f" long:"credentials-file" default:"./jokk.toml"`
}

type KafkaSettings struct {
	Host     string `toml:"host"`
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type JokkConfig struct {
	KafkaSettings map[string]KafkaSettings `toml:"kafka"`
	kafkaConfig
}

func main() {
	log := logrus.New()
	log.Info("welcome to Jokk")

	var args Args
	var parser = flags.NewParser(&args, flags.Default)
	if _, err := parser.Parse(); err != nil {
		switch flagsErr := err.(type) {
		case *flags.Error:
			if flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			}
			if args.Version {
				printVersion()
				os.Exit(0)
			}
			log.Error("error with command line argument: ", err)
			os.Exit(1)
		default:
			if args.Version {
				printVersion()
				os.Exit(0)
			}
			log.Error("error with command line argument: ", err)
			os.Exit(1)
		}
	}

	if args.Version {
		printVersion()
		os.Exit(0)
	}

	jokkConfig := JokkConfig{}
	err := jokkConfig.loadFromFile(args.CredentialsConfigFile)
	if err != nil {
		log.Error("could not load or parse configuration file: ", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)

	loop := true
	for loop {
		in, _ := reader.ReadString('\n')
		switch strings.Replace(strings.ToUpper(in), "\n", "", 1) {
		case "Q":
			loop = false
		case "H", "HELP":
			log.Infof("look at the README for supported commands or type 'q' to exit")
		case "LIST":
			log.Infof("listing Kafka stuffs...\n")
		default:
			// ignore the gibberish
		}
	}

	log.Info("bye!")
}

func (jc *JokkConfig) loadFromFile(file string) error {
	if file == "" {
		return fmt.Errorf("error: you must have a value for 'credentials-config-file'")
	}

	fp, err := hd.Expand(file)
	if err != nil {
		return fmt.Errorf("error: could not expand path(%s) to blob config file: %v", file, err)
	}

	_, err = toml.DecodeFile(fp, jc)
	if err != nil {
		return fmt.Errorf("error: failed to decode '%s': %v", fp, err)
	}

	return nil
}

func printVersion() {
	fmt.Printf(`
>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>
    Jokk - a view into the river of data
    
    Git SHA:                  %s
    Build Time:               %s
    Head Tags                 %s
    Head Version Tag:         %s
    Dirty build:              %s
    Describe Version:         %s
    Describe All:             %s
    Last Commit Date:         %s
    Last Commit Author:       %s
    Last Commit Author Email: %s
<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<
`, Sha1ver, BuildTime, HeadTags, HeadVersionTag, Dirty, DescribeVersion, DescribeAll, LastCommitDate, LastCommitAuthor, LastCommitEmail)
}

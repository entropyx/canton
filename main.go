package main

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"

	"github.com/jszwec/csvutil"
	log "github.com/sirupsen/logrus"
	"github.com/tebeka/selenium"
)

var m sync.Mutex

const (
	pool = 15
)

const (
	// These paths will be different on your system.
	seleniumPath    = "vendor/selenium-server-standalone-3.14.0.jar"
	geckoDriverPath = "vendor/chromedriver"
	port            = 3030
)

const (
	sourcePropiedades = "propiedades"
	sourceInmuebles   = "inmuebles"
)

type Result struct {
	Prices []string `goquery:"ol#searchResults li.results-item .item_price"`
	Size   string
}

type CSV struct {
	Col  string `csv:"col"`
	City string `csv:"city"`
}

type CSVResult struct {
	Col      string `csv:"col"`
	Price    string `csv:"price"`
	Size     string `csv:"size"`
	SizeType string `csv:"size type"`
}

func main() {
	var csvResults []*CSVResult
	service, err := selenium.NewChromeDriverService(geckoDriverPath, port)

	if err != nil {
		panic(err) // panic is used only as an example and is not otherwise recommended.
	}
	defer service.Stop()
	log.Info("Reading file...")
	var csvs []*CSV
	b, err := ioutil.ReadFile("data.csv")
	if err != nil {
		panic(err)
	}
	if err := csvutil.Unmarshal(b, &csvs); err != nil {
		panic(err)
	}
	wg := sync.WaitGroup{}
	// csvs = csvs[0:34]
	for i, csv := range csvs {
		go func(i int, csv *CSV) {
			log.Infof("%d / %d", i, len(csvs))
			caps := selenium.Capabilities{"browserName": "chrome"}
			wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", port))
			if err != nil {
				panic(err)
			}
			if err := wd.Get(csv.finalURL(sourceInmuebles)); err != nil {
				panic(err)
			}
			prices, err := wd.FindElements(selenium.ByClassName, "item__price")
			if err != nil {
				panic(err)
			}

			attrs, err := wd.FindElements(selenium.ByClassName, "item__attrs")
			if err != nil {
				panic(err)
			}
			log.Infof("%d properties were found\n", len(prices))

			for j, p := range prices {
				result := &CSVResult{
					Col: csv.Col,
				}
				attr := attrs[j]
				attrText, err := attr.Text()
				if err != nil {
					panic(err)
				}
				price, err := p.Text()
				if err != nil {
					panic(err)
				}
				regexSize, err := regexp.Compile("([0-9])+")
				if err != nil {
					panic(err)
				}
				b := regexSize.Find([]byte(attrText))
				result.Size = string(b)
				regexSizeType, err := regexp.Compile("([a-z]){2,}\\s*")
				if err != nil {
					panic(err)
				}
				s := regexSizeType.FindAllString(attrText, -1)
				if len(s) > 0 {
					result.SizeType = s[0]
				}
				regexPrice, err := regexp.Compile("([0-9])+")
				if err != nil {
					panic(err)
				}
				s = regexPrice.FindAllString(price, -1)
				price = strings.Join(s, "")
				result.Price = price
				// log.Info("Col ", result.Col)
				// log.Info("Price ", result.Price)
				// log.Info("Size ", result.Size)
				// log.Info("SizeType ", result.SizeType)
				m.Lock()
				csvResults = append(csvResults, result)
				m.Unlock()
			}
			wd.Close()
			wg.Done()
		}(i, csv)
		// if r := len(csvs) - i; r < pool {
		// 	wg.Wait()
		// 	wg.Add(1)
		// }
		if i%pool == 0 {
			log.Info("waiting...")
			wg.Wait()
			wg.Add(pool)
		}

	}
	data, err := csvutil.Marshal(&csvResults)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile("vendor/result.csv", data, 664); err != nil {
		panic(err)
	}
	log.Info("Bye!")
}

func (c *CSV) path(source string) string {
	col := format(c.Col)
	city := format(c.City)
	switch source {
	case sourceInmuebles:
		return fmt.Sprintf("%s-%s", col, city)
	case sourcePropiedades:
		return fmt.Sprintf("%s-%s", col, city)
	}
	return ""
}

func (c *CSV) finalURL(source string) string {
	switch source {
	case sourceInmuebles:
		return fmt.Sprintf("https://inmuebles.metroscubicos.com/oficinas/renta/%s", c.path(source))
	case sourcePropiedades:
		return fmt.Sprintf("https://propiedades.com/%s/comercial-renta", c.path(source))
	default:
		panic("unkown source")
	}
	return ""
}

func format(s string) string {
	return strings.Replace(strings.ToLower(s), " ", "-", -1)
}

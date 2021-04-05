package main

import (
	"encoding/json"
	"fmt"
	"github.com/aymerick/raymond"
	"github.com/icewind666/html-to-excel-renderer/src/config"
	"github.com/icewind666/html-to-excel-renderer/src/generator"
	"github.com/icewind666/html-to-excel-renderer/src/helpers"
	"github.com/icewind666/html-to-excel-renderer/src/types"
	"github.com/jbowtie/gokogiri"
	"github.com/jbowtie/gokogiri/xml"
	"github.com/jbowtie/gokogiri/xpath"
	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	_ "image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
)


var (
	version = "1.1.5"
	date    = "04.04.2021"
	builtBy = "v.korennoj@medpoint24.ru"
)

// Search strings for html tags
var XpathTable = xpath.Compile(".//table")
var XpathThead = xpath.Compile(".//thead/tr")
var XpathTh = xpath.Compile(".//th")
var XpathTr = xpath.Compile("./tr")
var XpathTd = xpath.Compile(".//td")
var XpathImg = xpath.Compile(".//img")


var conf = config.New()

func main() {
	if err := godotenv.Load(); err != nil {
		log.Infoln("No separate .env file specified. Using values from environment")
	}

	conf = config.New()
	log.SetOutput(os.Stdout)
	logLevel,err := log.ParseLevel(conf.LogLevel)

	if err != nil {
		log.Warn("Cannot parse debug level, default to INFO")
	} else {
		log.SetLevel(logLevel)
	}

	if len(os.Args) < 4 {
		log.Errorln("Usage:", os.Args[0], "<hbs_template> <data_json> <output_excel_file>")
		log.Fatalln("Invalid command line arguments")
	}

	log.Infof("html-to-excel-renderer v%s, built at %s by %s", version, date, builtBy)

	debugOn := conf.DebugMode

	if debugOn {
		log.Infoln("Debug mode is ON")
	}

	batchSize := conf.BatchSize
	filename := os.Args[1]
	dataFilename := os.Args[2]
	outputFilename := os.Args[3]

	renderedHtml := applyHandlebarsTemplate(filename, dataFilename)

	log.Infoln("Rendering Handlebars.js template to html is done")
	PrintMemUsage()

	if debugOn {
		err := ioutil.WriteFile("rendered.html", []byte(renderedHtml), 0777)
		if err != nil {
			log.WithError(err).Infoln("Cant write debug log - rendered html file!")
		}
	}

	generateXlsxFile(renderedHtml, outputFilename, batchSize)
	PrintMemUsage()
	log.Infoln("All done")
}


func NewHtmlStyle() *types.HtmlStyle {
	return &types.HtmlStyle {
		TextAlign:         "",
		WordWrap:          false,
		Width:             0,
		Height:            0,
		BorderInheritance: false,
		BorderStyle:       false,
		FontSize:          0,
		IsBold:            false,
		Colspan:           0,
		VerticalAlign:     "",
	}
}


// Reads and unmarshalls json from file
func ReadJsonFile(jsonFilename string) map[string]interface{} {
	byteValue, _ := ioutil.ReadFile(jsonFilename)

	if byteValue == nil {
		log.Fatalf("File is empty? %s\n", jsonFilename)
	}

	var result map[string]interface{}

	err := json.Unmarshal(byteValue, &result)
	if err != nil {
		log.Fatalln("Error reading data json file! Can't deserialize json!")
	}

	return result
}


// PrintMemUsage outputs the current, total and OS memory being used. As well as the number
// of garage collection cycles completed.
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	log.Infof("Alloc = %v MiB, HeapAlloc = %v MiB, Sys = %v MiB", bToMb(m.Alloc),
		bToMb(m.HeapAlloc), bToMb(m.Sys))
}

// Converts bytes to human readable file size
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}


func getExcelizeGenerator() *generator.ExcelizeGenerator {
	return &generator.ExcelizeGenerator{
		OpenedFile:   nil,
		Filename:     "",
		CurrentSheet: "",
		CurrentCol:   0,
		CurrentRow:   0,
	}
}

// Parses given html and generates xslt file.
// File is generated by adding batches of batchSize to in on every iteration.
func generateXlsxFile(html string, outputFilename string, batchSize int) string {
	doc, err := gokogiri.ParseHtml([]byte(html))

	if err != nil {
		log.WithError(err).Fatalln("Parse html ERROR!")
	}

	tables, _ := doc.Root().Search(XpathTable)
	defer doc.Free()

	// creating excel excelizeGenerator
	excelizeGenerator := getExcelizeGenerator()
	excelFilename := fmt.Sprintf("%s", outputFilename)
	excelizeGenerator.Filename = excelFilename
	excelizeGenerator.CurrentCol = 1
	excelizeGenerator.CurrentRow = 1
	excelizeGenerator.Create()

	totalRows := 0
	currentSheetIndex := 0

	// Main cycle through all tables in file
	for i, table := range tables {
		// Create new sheet for each table. Name it with data-name from html attribute
		sheetName := table.Attr("data-name")

		if sheetName == "" {
			sheetName = fmt.Sprintf("DataSheet %d", i)
			log.Infof("Warning! No data-name in for table found. Used %s as sheet name\n", sheetName)
		}

		if currentSheetIndex == 0 {
			excelizeGenerator.SetSheetName("Sheet1", sheetName)
		} else {
			excelizeGenerator.AddSheet(sheetName)
		}

		excelizeGenerator.CurrentCol = 1
		excelizeGenerator.CurrentRow = 0

		// Get thead for table and create header in xlsx
		theadTrs, _ := table.Search(XpathThead)
		processHtmlTheadTag(theadTrs, excelizeGenerator)

		// Get all rows in html table
		rows, _ := table.Search(XpathTr)
		rowsProceeded := 0
		packSize := batchSize

		for rowsProceeded < len(rows) {
			processTableRows(rows, excelizeGenerator, rowsProceeded, packSize)
			rowsProceeded += packSize
		}

		totalRows += len(rows) // stored only for log output
		rows = nil // just for sure. prevent memory leak which was found during tests in 3rd party lib
		currentSheetIndex += 1
	}

	excelizeGenerator.Save(excelizeGenerator.Filename)

	log.Infof("Total rows done: %d", totalRows)
	return excelFilename
}


// Process all html table rows
// Starts with <th> table headers then goes over <tr> and <td> inside them.
func processTableRows(rows []xml.Node, generator *generator.ExcelizeGenerator, offset int, rowsNumber int) {
	if offset >= len(rows) {
		return // offset cant be greater than number of rows
	}

	if len(rows) < rowsNumber {
		rowsNumber = len(rows) // when less than one page
	}

	for i := offset; i <= (offset + rowsNumber - 1); i++ {
		if i >= len(rows) {
			break // we are done here
		}

		tr := rows[i]
		generator.AddRow()

		theadTrs, _ := tr.Search(XpathTh)
		generator.CurrentCol = 1

		// <th>
		for _, theadTh := range theadTrs {
			thStyle := theadTh.Attribute(StyleAttrName)
			cellValue := theadTh.Content()

			if thStyle != nil {
				style := ExtractStyles(thStyle)
				thColspan := theadTh.Attribute(ColspanAttrName)

				if thColspan != nil {
					style.Colspan,_ = strconv.Atoi(thColspan.Value())
				}

				generator.ApplyColumnStyle(style)
				generator.ApplyCellStyle(style)
			}

			// <img>
			// NOTE: is it valid to have img in th?)
			imgs, _ := theadTh.Search(XpathImg)

			if len(imgs) > 0 {
				for _, img := range imgs {
					addImageToCell(img, generator)
				}
			} else {
				if cellValue != "" {
					generator.SetCellValue(cellValue)
				}
			}

			generator.CurrentCol += 1
		}

		cells, _ := tr.Search(XpathTd)
		generator.CurrentCol = 1

		// table td cells
		for _, td := range cells {
			tdStyle := td.Attribute("style")
			cellValue := td.Content()

			if tdStyle != nil {
				cellStyle := ExtractStyles(tdStyle)
				tdColspan := td.Attribute(ColspanAttrName)

				if tdColspan != nil {
					cellStyle.Colspan,_ = strconv.Atoi(tdColspan.Value())
				}

				generator.ApplyCellStyle(cellStyle)
			}

			imgs, _ := td.Search(XpathImg)

			if len(imgs) > 0 {
				for _, img := range imgs {
					addImageToCell(img, generator)
				}
			} else {
				if cellValue != "" {
					generator.SetCellValue(cellValue)
				}
			}

			generator.CurrentCol += 1
		}

		trStyle := tr.Attribute("style")

		// Apply row style if present
		if trStyle != nil {
			styleExtracted := ExtractStyles(trStyle)
			generator.ApplyRowStyle(styleExtracted)
		}
	}
}


func addImageToCell(img xml.Node, generator *generator.ExcelizeGenerator) {
	imgSrc := img.Attribute("src")
	imgAlt := img.Attribute("alt")

	// If file exist - set image to cell
	if _, err := os.Stat(imgSrc.Value()); os.IsNotExist(err) {
		if err != nil {
			log.WithError(err).Errorln("Cant access image file")
		}
		generator.SetCellValue(imgAlt.Value())
	}

	currentCellCoords, errCoords := generator.GetCoords()

	if errCoords != nil {
		log.WithError(errCoords).Errorln(errCoords)
	}

	errAdd := generator.OpenedFile.AddPicture(generator.CurrentSheet,
		currentCellCoords,
		imgSrc.Value(),
		`{"autofit":true, "lock_aspect_ratio": true, "positioning": "oneCell"}`)
	if errAdd != nil {
		log.Printf(errAdd.Error())
	}
}


// Process thead tag (thead->tr + thead->tr->th). Apply column styles. Apply cell styles
func processHtmlTheadTag(theadTrs []xml.Node, generator *generator.ExcelizeGenerator) {
	for _, theadTr := range theadTrs {
		generator.AddRow()
		theadTrThs, _ := theadTr.Search(XpathTh) // search for <th>
		colIndex := 1

		for _, theadTh := range theadTrThs { // for each <th> in <tr>
			thStyle := theadTh.Attribute(StyleAttrName)

			style := ExtractStyles(thStyle)
			thColspan := theadTh.Attribute(ColspanAttrName)

			if thColspan != nil {
				style.Colspan, _ = strconv.Atoi(thColspan.Value())
			}

			content := theadTh.Content()

			if content != "" {
				if style.CellValueType == FloatValueType {
					floatContent,err := strconv.ParseFloat(content, 64)

					if err != nil {
						log.WithError(err).Error("Cant parse cell type")
					}

					generator.SetCellFloatValue(floatContent)
				}

				if style.CellValueType == StringValueType {
					generator.SetCellValue(content)
				}
			}

			if style != nil {
				generator.ApplyColumnStyle(style)
				generator.ApplyCellStyle(style)
			}

			colIndex++
		}

		thStyle := theadTr.Attribute(StyleAttrName)

		if thStyle != nil {
			rowStyle := ExtractStyles(thStyle)
			thColspan := thStyle.Attribute(ColspanAttrName)

			if thColspan != nil {
				rowStyle.Colspan, _ = strconv.Atoi(thColspan.Value())
			}
			if rowStyle != nil {
				generator.ApplyRowStyle(rowStyle)
			}
		}
	}
}


// Apply Handlebars template to json data in dataFilename file.
// Note: takes content by "data" key
func applyHandlebarsTemplate(templateFilename string, dataFilename string) string {
	jsonCtx := ReadJsonFile(dataFilename)
	tpl, err := raymond.ParseFile(templateFilename)

	if err != nil {
		log.WithError(err).Fatalf("Error while parsing template %s \n", templateFilename)
	}

	// register helpers
	registerAllHelpers(tpl)
	data := jsonCtx
	result, err := tpl.Exec(data)

	if err != nil {
		log.WithError(err).Fatalf("Error applying template %s to json file %s", templateFilename, dataFilename)
	}

	return result
}

func registerAllHelpers(template *raymond.Template)  {
	template.RegisterHelper("math", helpers.MathHelper)
	template.RegisterHelper("key", helpers.KeyHelper)
	template.RegisterHelper("zeroIntHelper", helpers.ZeroIntHelper)
	template.RegisterHelper("percentHelper", helpers.PercentHelper)
	template.RegisterHelper("inspectionTimeHelper", helpers.InspectionTimeHelper)
	template.RegisterHelper("dashHelper", helpers.DashHelper)
	template.RegisterHelper("pressureHelper", helpers.PressureHelper)
	template.RegisterHelper("allowHelper", helpers.AllowHelper)
	template.RegisterHelper("upper", helpers.UpperHelper)
	template.RegisterHelper("ifnull", helpers.IfNullHelper)
	template.RegisterHelper("isAfterBeforeSheet", helpers.IsAfterBeforeSheetHelper)
	template.RegisterHelper("summarize", helpers.SummarizeHelper)
	template.RegisterHelper("lineSumRows", helpers.LineSumRowsHelper)
	template.RegisterHelper("faceIdNotFoundName", helpers.FaceIdNotFoundNameHelpder)
}



// Returns parsed style struct
func ExtractStyles(node *xml.AttributeNode) *types.HtmlStyle {
	if node == nil {
		return NewHtmlStyle()
	}

	styleStr := node.Content()
	entries := strings.Split(styleStr, ";")
	resultStyle := NewHtmlStyle()

	for _, e := range entries {
		if e != "" {
			parts := strings.Split(e, ":")

			if len(parts) < 2 {
				continue
			}

			value := strings.Trim(parts[1], " ")
			attr := strings.Trim(parts[0], " ")

			switch attr {
			case ColspanAttrName:
				resultStyle.Colspan, _ = strconv.Atoi(value)

			case TextAlignStyleAttr:
				resultStyle.TextAlign = value

			case WordWrapStyleAttr:
				resultStyle.WordWrap = value == BreakWordWrapStyleAttrValue

			case WidthStyleAttr:
				widthEntry := strings.Trim(value, " px")
				widthInt, _ := strconv.Atoi(widthEntry)
				translatedWidth := float64(widthInt) * conf.SizeTransform.PxToExcelWidthMultiplier
				resultStyle.Width = translatedWidth

			case MinWidthStyleAttr:
				if resultStyle.Width <= 0 {
					widthEntry := strings.Trim(value, " px")
					widthInt, _ := strconv.Atoi(widthEntry)
					translatedWidth := float64(widthInt) * conf.SizeTransform.PxToExcelWidthMultiplier
					resultStyle.Width = translatedWidth
				}
			case MaxWidthStyleAttr:
				if resultStyle.Width <= 0 {
					widthEntry := strings.Trim(value, " px")
					widthInt, _ := strconv.Atoi(widthEntry)
					translatedWidth := float64(widthInt) * conf.SizeTransform.PxToExcelWidthMultiplier
					resultStyle.Width = translatedWidth
				}

			case HeightStyleAttr:
				heightEntry := strings.Trim(value, " px")
				heightInt, _ := strconv.Atoi(heightEntry)
				translatedHeight := float64(heightInt) * conf.SizeTransform.PxToExcelHeightMultiplier
				resultStyle.Height = translatedHeight

			case MinHeightStyleAttr:
				if resultStyle.Height <= 0 {
					heightEntry := strings.Trim(value, " px")
					heightInt, _ := strconv.Atoi(heightEntry)
					translatedHeight := float64(heightInt) * conf.SizeTransform.PxToExcelHeightMultiplier
					resultStyle.Height = translatedHeight
				}
			case MaxHeightStyleAttr:
				if resultStyle.Height <= 0 {
					heightEntry := strings.Trim(value, " px")
					heightInt, _ := strconv.Atoi(heightEntry)
					translatedHeight := float64(heightInt) * conf.SizeTransform.PxToExcelHeightMultiplier
					resultStyle.Height = translatedHeight
				}
			case BorderStyleAttr:
				resultStyle.BorderStyle = value == BorderStyleAttrValue

			case BorderInheritanceStyleAttr:
				resultStyle.BorderStyle = value == BorderInheritanceStyleAttrValue

			case FontSizeStyleAttr:
				widthEntry := strings.Trim(value, " px")
				sz,_ := strconv.Atoi(widthEntry)
				resultStyle.FontSize = float64(sz)

			case FontWeightStyleAttr:
				resultStyle.IsBold = strings.Contains(value, "bold")

			case TextVerticalAlignStyleAttr:
				if value == "middle" {
					value = "center" // excelize lib dont understand middle :) center works fine
				}
				resultStyle.VerticalAlign = value
			case ValueTypeAttrName:
				cellType := types.ValueType(value)
				switch cellType { // filter only supported types
				case FloatValueType:
				case StringValueType:
				case DateValueType:
				case BooleanValueType:
					resultStyle.CellValueType = cellType
				}
			}

		}
	}
	return resultStyle
}

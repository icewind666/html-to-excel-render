 # html-to-excel-renderer

 ## Install via npm

Npm package contains 2 scripts - postinstall and preuninstall - they are downloading binaries and use go-npm 
(https://www.npmjs.com/package/go-npm) to install them

` npm install -g go-html-to-excel-renderer`
 


 ## Install via downloading binaries
 On releases page choose your binary (https://github.com/icewind666/html-to-excel-renderer/releases)
 
 Then download it, unpack and put to accessible location
 (below is an example for linux_amd64)
 
 `wget https://github.com/icewind666/html-to-excel-renderer/releases/download/v1.1.5/html-to-excel-renderer_v1.1.5_linux_x86_64.tar.gz`

 `tar -xvf html-to-excel-renderer_v1.1.5_linux_x86_64.tar.gz`

 `sudo mv html-to-excel-renderer_v1.1.5_linux_x86_64/html-to-excel-renderer /usr/bin`


## Install from source building

 Dependencies: 
 `libxml2-dev`, `libc6-dev`
(`sudo apt-get install libxml2-dev libc6-dev`)
 

`go build -o html-to-excel-renderer github.com/icewind666/html-to-excel-renderer/src/main `

Run command be like:

`html-to-excel-renderer <template> <data> <output>`

Example: `html-to-excel-renderer template.hbs data.json result.xslx`


**template** - handlebars template file (hbs)

**data** - report data file (json)

**output** - report output


## Environment settings

| Variable      | Description   |
| ------------- |:-------------|
| BatchSize     | Number of rows to process on one iteration. Applies to each sheet |
| PxToExcelWidthMultiplier     | Multiplier used to map pixels in html to width in excel |
| PxToExcelHeightMultiplier     | Multiplier used to map pixels in html to height in excel |
| DebugMode     | Enables writing temporary html file with rendered content. File is NOT removed but overwritten every run |
| GoRenderLogLevel     | Log level. Default is info |

## 3rd party libs

For Handlebars.js template rendering:
**https://github.com/aymerick/raymond**

 For html parsing:
 **https://github.com/jbowtie/gokogiri**
 
 For XLSX generation Excel:
 **https://github.com/360EntSecGroup-Skylar/excelize/v2**

Loading env variables:
**https://github.com/joho/godotenv**

For logs:
**https://github.com/sirupsen/logrus**
 
 
 

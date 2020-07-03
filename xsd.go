package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"tools/pkg/util/logd"

	"golang.org/x/net/html/charset"

	"github.com/golang-collections/collections/stack"
	"github.com/lestrrat/go-libxml2"
	"github.com/lestrrat/go-libxml2/xsd"
)

// referer :https://www.w3school.com.cn/schema/index.asp
type XSDSchema struct {
    XMLName xml.Name `xml:"http://www.w3.org/2001/XMLSchema schema"`
    TargetNS string `xml:"targetNamespace,attr,omitempty"`
    AttrFromDefault string `xml:"attributeFormDefault,attr,omitempty"`
    ElemFromDefault string `xml:"elementFormDefault,attr,omitempty"`
    Import []*XSDImport `xml:"import,omitempty"`
    Elem *XSDElement `xml:"element,omitempty"`
    validSchema *xsd.Schema
}

type XSDImport struct {
    SchemaLocation string `xml:"schemaLocation,attr"`
    NameSpace string `xml:"namespace,attr"`
    schema *XSDSchema
}
type XSDElement struct {
    Name string `xml:"name,attr,omitempty"`
    XmlNS xml.`xml:"attr,omitempty"`
    Ref string `xml:"ref,attr,omitempty"`
    Type string `xml:"type,attr,omitempty"`
    MinOccurs string `xml:"minOccurs,attr,omitempty"` 
    MaxOccurs string `xml:"maxOccurs,attr,omitempty"` //unbounded
    ComplexType *XSDComplexType `xml:"complexType,omitempty"`
}

type XSDComplexType struct {
    SimpleContent *XSDSimpleContent `xml:"simpleContent,omitempty"`
    Sequence *XSDSequence `xml:"sequence,omitempty"` // xs:sequence -> all 
    Attr []*XSDAttribute `xml:"attribute,omitempty"`
    Mixed string `xml:"mixed,attr,omitempty"`
}

type XSDSimpleContent struct {
    Extension XSDExtension `xml:"extension"`
}

type XSDExtension struct {
    Base string `xml:"base,attr"` // xs:string...
    Attr []*XSDAttribute `xml:"attribute,omitempty"`
}

type XSDAttribute struct {
    Name string `xml:"name,attr,omitempty"`
    //XmlNS string `xml:"xmlns:xs,attr"`
    Type string `xml:"type,attr,omitempty"`
    Use string `xml:"use,attr,omitempty"`
}
type XSDChoice struct {
    MaxOccurs string `xml:"maxOccurs,attr,omitempty"` //unbounded
    Elem []*XSDElement `xml:"element,omitempty"`
}
type XSDSequence struct {
    Choice XSDChoice `xml:"choice,omitempty"`
}

func XSDValueType(val string) string {
    // TODO
/*xs:string
xs:decimal
xs:integer
xs:boolean
xs:date
xs:time*/
    val = strings.TrimSpace(val)
    if val == "" || val == "\r" || val == "\n" || val == " " || val == "\t" {
        return ""
    }
    return "string" //TODO Fixed to return val, type process in proxyd
}

func (elem *XSDElement) SetChild(child *XSDElement) {
    child.MinOccurs = "0"
    if elem.ComplexType == nil {
        elem.ComplexType = &XSDComplexType{Sequence: &XSDSequence{Choice: XSDChoice{Elem: []*XSDElement{child}, MaxOccurs: "unbounded"}}}
    } else if elem.ComplexType.Sequence == nil {
        elem.ComplexType.Sequence = &XSDSequence{Choice: XSDChoice{Elem: []*XSDElement{child}, MaxOccurs: "unbounded"}}      
    } else {
        elems := elem.ComplexType.Sequence.Choice.Elem
        find := false
        for _, e := range elems {
            if e.Name == child.Name {
                if child.ComplexType != nil && child.ComplexType.Sequence != nil {
                    if e.ComplexType != nil && e.ComplexType.Sequence != nil {
                        e.ComplexType.Sequence.Merge(child.ComplexType.Sequence)
                    } else {
                        e.ComplexType = child.ComplexType
                    }
                }
                e.MaxOccurs = "unbounded"
                find = true
            }
        }
        if !find {
            elem.ComplexType.Sequence.Choice.Elem = append(elem.ComplexType.Sequence.Choice.Elem, child)
        }
    }
}
func (ep *XSDElement) ConvertToSimpleContent() {
    if ep.ComplexType != nil { // convert to simpleContent
        if ep.Type != "" && ep.ComplexType.Sequence == nil && len(ep.ComplexType.Attr) > 0 {
            ep.ComplexType.SimpleContent = &XSDSimpleContent{}
            ep.ComplexType.SimpleContent.Extension.Attr = append(
                ep.ComplexType.SimpleContent.Extension.Attr, ep.ComplexType.Attr...
            )
            ep.ComplexType.Attr = nil
            ep.ComplexType.SimpleContent.Extension.Base = ep.Type
            ep.Type = ""
        }
    }
}

func (ep *XSDElement) ComplexeTypeMixed() {
    if ep.Type != "" && ep.ComplexType != nil {
        ep.Type = ""
        ep.ComplexType.Mixed = "true"
    }
}

func GenXSDFromDecoder(decoder *xml.Decoder, root xml.Token, idx int) (*XSDSchema, xml.Token){
    var t xml.Token
    var ep *XSDElement
    var ns string
    var ipt []*XSDImport
    elemStack := stack.New()
    
    for t = root; t != nil; {
        switch token := t.(type) {
        case xml.StartElement:
            ep = &XSDElement{Name: token.Name.Local}
            if ns == "" {
                ns = token.Name.Space
            } else if ns != token.Name.Space {

                ep.XmlNS = xml.Attr{Value: token.Name.Space, Name: xml.Name{Space:"xmlns", Local:"q1"}}
                ep.Ref = "q1:"+token.Name.Local
                ep.Name = ""
                innerSchema, prev := GenXSDFromDecoder(decoder, t, idx+1)
                if innerSchema == nil {
                    logd.Error("create import schema failed!\n")
                    return nil, nil
                }
                ipt = append(ipt, &XSDImport{SchemaLocation: fmt.Sprintf("%d.xsd", idx),
                                             NameSpace: innerSchema.TargetNS,
                                             schema: innerSchema,})

                elemStack.Push(ep)
                t = prev
                goto next
            } 
            for _, attr := range token.Attr {
                if attr.Name.Space == "xmlns" || attr.Name.Local == "xmlns" {
                    continue
                }
                na := &XSDAttribute{Name: attr.Name.Local,
                                    Type: XSDValueType(attr.Value),
                                    Use: "optional"}
                if ep.ComplexType == nil {
                    ep.ComplexType = &XSDComplexType{Attr: []*XSDAttribute{na}}
                } else {
                    ep.ComplexType.Attr = append(ep.ComplexType.Attr, na)
                }
            }
            elemStack.Push(ep)
        case xml.EndElement:
            ep = elemStack.Pop().(*XSDElement)
            if ep == nil {
                panic(errors.New("XML Parse falt!!!"))
            }
            ep.ConvertToSimpleContent()
            ep.ComplexeTypeMixed()

            top := elemStack.Peek()
            if top == nil {
                goto end
            }
            top.(*XSDElement).SetChild(ep)
        case xml.CharData:
            if elemStack.Len() > 0 {
                ep := elemStack.Peek().(*XSDElement)
                ep.Type = XSDValueType(string([]byte(token)))
            }
        default:
        }
        t, _ = decoder.Token()
        next:
    }

end:
    return &XSDSchema{
                    TargetNS: ns,
                    AttrFromDefault: "unqualified",
                    ElemFromDefault: "qualified",
                    Elem: ep,
                    Import: ipt}, t 
}

func writeFileToTempDir(s *XSDSchema, dir string, idx int) (string, error) {
    for _, v := range s.Import {
        _, err := writeFileToTempDir(v.schema, dir, idx+1)
        if err != nil {
            logd.Error(err.Error())
            return "", err
        }
    }
    xsdStr := s.Marshal()
    filename := filepath.Join(dir, fmt.Sprintf("%d.xsd", idx))
    err := ioutil.WriteFile(filename, []byte(xsdStr), 0666)
    if err != nil {
        logd.Error(err.Error())
        return "", err
    }
    return filename, nil
}

func (self *XSDSchema)ValidSchemaInit() error {
    dir, err := ioutil.TempDir("", "apid")
    if err != nil {
        logd.Error(err.Error())
        return err
    }
    defer os.RemoveAll(dir)
    fn, err := writeFileToTempDir(self, dir, 0)
    if err != nil {
        logd.Error(err.Error())
        return err
    }

    s, err := xsd.ParseFromFile(fn)
    if err != nil {
        logd.Error(err.Error())
        return err
    }
    self.validSchema = s

    return nil
}

func (self *XSDSchema)ValidXML(inXML []byte) bool {
    if self.validSchema == nil {
        logd.Error("XSDSchema Not Inital...\n")
        return false
    }
    d, err := libxml2.Parse(inXML)
    if err != nil {
        logd.Error(err.Error())
        return false
    }
    defer d.Free()

    if err := self.validSchema.Validate(d); err != nil {
        for _, e := range err.(xsd.SchemaValidationError).Errors() {
            println(e.Error())
        }
        return false
    }

    return true
}
func GenXSDFromXML(rawXML []byte) *XSDSchema {

    iReader := strings.NewReader(string(rawXML))
    decoder := xml.NewDecoder(iReader)
    decoder.CharsetReader = charset.NewReaderLabel
    
    token, err := decoder.Token()
    if err != nil {
        logd.Error(err.Error())
        return nil
    }
    ret, _ := GenXSDFromDecoder(decoder, token, 1)
    err = ret.ValidSchemaInit()
    if err != nil {
        panic(err)
    }
    return ret
}

const (
    XSDFWBFlag = `<!-- Created with FWB API Discover -->`
)
func XSDSchemaUnmarshalAll(schemaSet []string, idx int) (*XSDSchema, error) {
    var xsdRet XSDSchema
    err := xml.Unmarshal([]byte(schemaSet[idx]), &xsdRet)
    if err != nil {
        panic(err)
    }

    for _, v := range xsdRet.Import {
        s, err := XSDSchemaUnmarshalAll(schemaSet, idx+1)
        if err != nil {
            panic(err)
        }
        v.schema = s
    }

    return &xsdRet, nil
}
func (xsd *XSDSchema) MarshalAll() []string {
    var result []string
    scm, err := xml.Marshal(xsd)
    //scm, err := xml.MarshalIndent(xsd, "xs", "")
    if err != nil {
        logd.Error(err.Error())
        return result
    }
    result = append(result, xml.Header+XSDFWBFlag+string(scm))

    for _, v := range xsd.Import {
        ret := v.schema.MarshalAll()
        result = append(result, ret...)
    }
    return result
}
func (xsd *XSDSchema) Marshal() string {
    scm, err := xml.Marshal(xsd)
    //scm, err := xml.MarshalIndent(xsd, "xs", "")
    if err != nil {
        logd.Error(err.Error())
        return ""
    }
    return xml.Header+XSDFWBFlag+string(scm)
}

func (s *XSDSequence) Merge(ss *XSDSequence) {
    var newAdd []*XSDElement
    for _, sse := range ss.Choice.Elem {
        find := false

        for _, se := range s.Choice.Elem {
            if sse.Name == se.Name {
                se.Merge(sse)
                find = true
            }
        }

        if !find {
            newAdd = append(newAdd, sse)
        }
    }

    s.Choice.Elem = append(s.Choice.Elem, newAdd...)
}

func mergeXSDAttr(dst *[]*XSDAttribute, src []*XSDAttribute) {
    var add []*XSDAttribute 
    for _, s := range src {
        find := false
        for _, v := range *dst {
            if v.Name == s.Name {
                find = true
                break
            }
        }
        if !find {
            add = append(add, s)
        }
    }

    *dst = append(*dst, add...)
}

func (s *XSDSimpleContent) Merge(ss *XSDSimpleContent) {

    if s.Extension.Base != ss.Extension.Base {
        mergeDataType(&s.Extension.Base, ss.Extension.Base)
    }

    mergeXSDAttr(&s.Extension.Attr, ss.Extension.Attr)
    //s.Extension.Attr = append(s.Extension.Attr, ss.Extension.Attr...)
}

func (c *XSDComplexType) Merge(cc *XSDComplexType) {
    if c.SimpleContent != nil && cc.SimpleContent != nil {
        c.SimpleContent.Merge(cc.SimpleContent)
    } 

    if c.Sequence != nil {
        if cc.Sequence != nil {
            c.Sequence.Merge(cc.Sequence)
        }
    } else if cc.Sequence != nil {
        c.Sequence = cc.Sequence
    }

    mergeXSDAttr(&c.Attr, cc.Attr)
    //c.Attr = append(c.Attr, cc.Attr...)
}

func (e *XSDElement) Merge(ee *XSDElement) {
    if e.Name != ee.Name {
        return
    }
    if e.ComplexType != nil {
        if ee.ComplexType != nil {
            e.ComplexType.Merge(ee.ComplexType)
        } // else no process
    } else if ee.ComplexType != nil {
        e.ComplexType = ee.ComplexType
    }
}
func (x *XSDSchema) Merge(xx *XSDSchema) {
    x.Elem.Merge(xx.Elem)
}

func ValidXsdWithXml(xsdRaw []byte, xmlRaw []byte) bool {
    s, err := xsd.Parse(xsdRaw)
    if err != nil {
        logd.Error(string(xsdRaw))
        panic(err)
    }
    defer s.Free()

    d, err := libxml2.Parse(xmlRaw)
    if err != nil {
        panic(err)
    }
    defer d.Free()

    if err := s.Validate(d); err != nil {
        for _, e := range err.(xsd.SchemaValidationError).Errors() {
            println(e.Error())
        }
        return false
    }

    return true
}
func TestGenXSD(data []byte, msg string) {
    xsdSchema := GenXSDFromXML([]byte(data))
    fmt.Println(xsdSchema.Marshal())
    if xsdSchema.ValidXML(data){
        fmt.Printf("%s Test Pass\n", msg)
    } else {
        fmt.Printf("%s Test Faild\n", msg)
    }

    allxsd := xsdSchema.MarshalAll()
    fmt.Println("XSD:", allxsd)

    xsds, err := XSDSchemaUnmarshalAll(allxsd, 0)
    if err != nil {
        panic(err)
    }
    xsds.ValidSchemaInit()
    if xsds.ValidXML(data){
        fmt.Printf("%s Second Test Pass\n", msg)
    } else {
        fmt.Printf("%s Second Test Faild\n", msg)
    }
}

func TestXSDUnmarshal() {
    //data := []byte(`<xs:schema attributeFormDefault="unqualified" elementFormDefault="qualified" xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="response"><xs:complexType><xs:sequence><xs:element name="snooze" minOccurs="0"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="alarms" type="xs:string" use="optional"></xs:attribute></xs:extension></xs:simpleContent></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element></xs:schema>`)
    data := []byte(`<?xml version="1.0" encoding="utf-8"?><!-- Created with FWB API Discover --><schema xmlns="http://www.w3.org/2001/XMLSchema" xmlns:xs="http://www.w3.org/2001/XMLSchema" attributeFormDefault="unqualified" elementFormDefault="qualified"><element xmlns="http://www.w3.org/2001/XMLSchema" name="letter" xmlns:xs="http://www.w3.org/2001/XMLSchema" type="xs:string"></element></schema>`)
    var v XSDSchema

    err := xml.Unmarshal(data, &v)
    if err != nil {
        panic(err)
    }

}

func XSDUnitTest() {
    xmlTestTwoXSD := []byte(`<?xml version="1.0" encoding="utf-8"?>
<!-- Created with Liquid Technologies Online Tools 1.0 (https://www.liquid-technologies.com) -->
<soap:Envelope  xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Body>
    <CheckAuthorizedUserResponse  xmlns="http://www.oraycn.com/">
      <CheckAuthorizedUserResult>true</CheckAuthorizedUserResult>
    </CheckAuthorizedUserResponse>
  </soap:Body>
</soap:Envelope>`)
    TestGenXSD([]byte(xmlTestTwoXSD), "XML Test Two XSD")
    xmlTestSimpleContentBug := 
`<cross-domain-policy>
    <allow-access-from domain="*.baidu.com" secure="false"/>
</cross-domain-policy>`
    TestGenXSD([]byte(xmlTestSimpleContentBug), "XML Test Simple content Bug")
xmlTestBugData := 
    `<?xml version="1.0" encoding="UTF-8"?>
<methodResponse>
  <fault>
    <value>
      <struct>
        <member>
          <name>faultCode</name>
          <value><int>403</int></value>
        </member>
        <member>
          <name>faultString</name>
          <value><string>Your IP (113.111.80.220) has been flagged for potential security violations.</string></value>
        </member>
      </struct>
    </value>
  </fault>
</methodResponse>`
    TestGenXSD([]byte(xmlTestBugData) , "XML Test BUG")

    TestXSDUnmarshal()
simpleTestXMLData := 
`<letter>
    Hi, Dear Mr.
</letter>`
    TestGenXSD([]byte(simpleTestXMLData) , "Simple XML")

simpleContentTestXMLData := 
`<food type="dessert">
    Ice cream
</food>`
    TestGenXSD([]byte(simpleContentTestXMLData) , "Simple Content XML")

complexTypeTestXMLData := 
`<letter>
    <name>John Smith</name>
    <orderid>1032</orderid>
    <shipdate>2001-07-13</shipdate>
</letter>`
    TestGenXSD([]byte(complexTypeTestXMLData) , "Complex Type XML")

complexTypeMixedTestXMLData := 
`<letter>
    Dear Mr.<name>John Smith</name>.
    Your order <orderid>1032</orderid>
    will be shipped on <shipdate>2001-07-13</shipdate>.
</letter>`
    TestGenXSD([]byte(complexTypeMixedTestXMLData) , "Complex Type Mixed XML")

complexTypeMulitLevelTestXMLData := 
`<Person>
    <shoesize country="france">35</shoesize>
    <FullName>Grace R. Emlin</FullName>
    <Company>Example Inc.</Company>
    <Email where="home">
        <Addr>gre@example.com</Addr>
    </Email>
    <Email where='work'>
        <Addr>gre@work.com</Addr>
    </Email>
    <Group>
        <Value>Friends</Value>
        <Value>Squash</Value>
    </Group>
    <City>Hanga Roa</City>
    <State>Easter Island</State>
</Person>`
    TestGenXSD([]byte(complexTypeMulitLevelTestXMLData) , "Complex Type Mulit Level XML")

complexTypeChoiceTestXMLData := 
`<person>
<user>
  <firstname></firstname>
  <lastname></lastname>
</user>
<user>
  <firstname></firstname>  
  <midlename></midlename>
</user>
<user> 
  <firstname></firstname> 
  <lastname></lastname>
  <midlename></midlename>
</user>
</person>`
    TestGenXSD([]byte(complexTypeChoiceTestXMLData) , "Complex Type Choice XML")

    

}
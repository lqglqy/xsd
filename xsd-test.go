package xsd

import (
	"encoding/xml"
	"fmt"
)

func TestGenXSD(data []byte, msg string) {
	xsdSchema := GenXSDFromXML([]byte(data))
	fmt.Println(xsdSchema.Marshal())
	if xsdSchema.ValidXML(data) {
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
	if xsds.ValidXML(data) {
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
	TestGenXSD([]byte(xmlTestBugData), "XML Test BUG")

	TestXSDUnmarshal()
	simpleTestXMLData :=
		`<letter>
    Hi, Dear Mr.
</letter>`
	TestGenXSD([]byte(simpleTestXMLData), "Simple XML")

	simpleContentTestXMLData :=
		`<food type="dessert">
    Ice cream
</food>`
	TestGenXSD([]byte(simpleContentTestXMLData), "Simple Content XML")

	complexTypeTestXMLData :=
		`<letter>
    <name>John Smith</name>
    <orderid>1032</orderid>
    <shipdate>2001-07-13</shipdate>
</letter>`
	TestGenXSD([]byte(complexTypeTestXMLData), "Complex Type XML")

	complexTypeMixedTestXMLData :=
		`<letter>
    Dear Mr.<name>John Smith</name>.
    Your order <orderid>1032</orderid>
    will be shipped on <shipdate>2001-07-13</shipdate>.
</letter>`
	TestGenXSD([]byte(complexTypeMixedTestXMLData), "Complex Type Mixed XML")

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
	TestGenXSD([]byte(complexTypeMulitLevelTestXMLData), "Complex Type Mulit Level XML")

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
	TestGenXSD([]byte(complexTypeChoiceTestXMLData), "Complex Type Choice XML")

}

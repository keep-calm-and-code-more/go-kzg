package kate

import (
	"testing"
)

func TestFFTRoundtrip(t *testing.T) {
	fs := NewFFTSettings(4)
	data := make([]Big, fs.maxWidth, fs.maxWidth)
	for i := uint64(0); i < fs.maxWidth; i++ {
		asBig(&data[i], i)
	}
	coeffs, err := fs.FFT(data, false)
	if err != nil {
		t.Fatal(err)
	}
	res, err := fs.FFT(coeffs, true)
	if err != nil {
		t.Fatal(err)
	}
	for i := range res {
		if got, expected := &res[i], &data[i]; !equalBig(got, expected) {
			t.Errorf("difference: %d: got: %s  expected: %s", i, bigStr(got), bigStr(expected))
		}
	}
	t.Log("zero", bigStr(&ZERO))
	t.Log("zero", bigStr(&ONE))
}

func TestInvFFT(t *testing.T) {
	fs := NewFFTSettings(4)
	data := make([]Big, fs.maxWidth, fs.maxWidth)
	for i := uint64(0); i < fs.maxWidth; i++ {
		asBig(&data[i], i)
	}
	debugBigs("input data", data)
	res, err := fs.FFT(data, true)
	if err != nil {
		t.Fatal(err)
	}
	debugBigs("result", res)
	bigNumHelper := func(v string) (out Big) {
		bigNum(&out, v)
		return
	}
	expected := []Big{
		bigNumHelper("26217937587563095239723870254092982918845276250263818911301829349969290592264"),
		bigNumHelper("40905488090558605688319636812215252217941835718478251840326926365086504505065"),
		bigNumHelper("10037948829646534413413739647971946522809495755620173630072972432081702959148"),
		bigNumHelper("43571192877568624546930318420751319449039972945062659080199348274630726213098"),
		bigNumHelper("26217937587563095241456442667129809078233411015607690300436955584351971573760"),
		bigNumHelper("23495295218275555727033128776954731040973520495197797376593908347998044220817"),
		bigNumHelper("10037948829646534409948594821898294204033226224932430851802719963316340996140"),
		bigNumHelper("20829590431265536861492157516271359172322844207237904580180981500923098586768"),
		bigNumHelper("26217937587563095239723870254092982918845276250263818911301829349969290592256"),
		bigNumHelper("31606284743860653617955582991914606665367708293289733242422677199015482597744"),
		bigNumHelper("42397926345479656069499145686287671633657326275595206970800938736622240188372"),
		bigNumHelper("28940579956850634752414611731231234796717032005329840446009750351940536963695"),
		bigNumHelper("26217937587563095237991297841056156759457141484919947522166703115586609610752"),
		bigNumHelper("8864682297557565932517422087434646388650579555464978742404310425307854971414"),
		bigNumHelper("42397926345479656066034000860214019314881056744907464192530686267856878225364"),
		bigNumHelper("11530387084567584791128103695970713619748716782049385982276732334852076679447"),
	}
	for i := range res {
		if got := &res[i]; !equalBig(got, &expected[i]) {
			t.Errorf("difference: %d: got: %s  expected: %s", i, bigStr(got), bigStr(&expected[i]))
		}
	}
}

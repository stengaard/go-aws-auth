package awsauth

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCommonFunctions(t *testing.T) {
	Convey("Service and region should be properly extracted from host strings", t, func() {
		s1, r1 := serviceAndRegion("sqs.us-west-2.amazonaws.com")
		So(s1, ShouldEqual, "sqs")
		So(r1, ShouldEqual, "us-west-2")

		s2, r2 := serviceAndRegion("iam.amazonaws.com")
		So(s2, ShouldEqual, "iam")
		So(r2, ShouldEqual, "us-east-1")
	})

	Convey("SHA-256 hashes should be properly hex-encoded (base 16)", t, func() {
		input := "This is... Sparta!!"
		actual := hashSHA256(input)

		So(actual, ShouldEqual, expecting["Sha256Hash"])
	})

	Convey("Given a key and contents", t, func() {
		key := []byte("asdf1234")
		contents := "SmartyStreets was here"

		Convey("HMAC-SHA256 should be properly computed", func() {
			expected := []byte{65, 46, 186, 78, 2, 155, 71, 104, 49, 37, 5, 66, 195, 129, 159, 227, 239, 53, 240, 107, 83, 21, 235, 198, 238, 216, 108, 149, 143, 222, 144, 94}
			actual := hmacSHA256(key, contents)

			So(actual, ShouldResemble, expected)
		})
	})

	Convey("Strings should be properly concatenated with a delimiter", t, func() {
		So(concat("\n", "Test1", "Test2"), ShouldEqual, "Test1\nTest2")
		So(concat(".", "Test1"), ShouldEqual, "Test1")
		So(concat("\t", "1", "2", "3", "4"), ShouldEqual, "1\t2\t3\t4")
	})
}
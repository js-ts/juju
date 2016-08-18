// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package constraints_test

import (
	"regexp"

	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/constraints"
)

type validationSuite struct{}

var _ = gc.Suite(&validationSuite{})

type intersectionSuite struct {
	originalAttributeValues            []string
	additionalAttributeValues          []string
	validCons                          string
	invalidCons                        string
	errorBeforeIntersection            string
	errorForValidConsAfterIntersection string
	errorAfterIntersection             string
}

var _ = gc.Suite(&intersectionSuite{})

var validationTests = []struct {
	cons        string
	unsupported []string
	vocab       map[string][]interface{}
	reds        []string
	blues       []string
	err         string
}{
	{
		cons: "root-disk=8G mem=4G arch=amd64 cpu-power=1000 cpu-cores=4",
	},
	{
		cons:        "root-disk=8G mem=4G arch=amd64 cpu-power=1000 cpu-cores=4 tags=foo",
		unsupported: []string{"tags"},
	},
	{
		cons:        "root-disk=8G mem=4G arch=amd64 cpu-power=1000 cpu-cores=4 instance-type=foo",
		unsupported: []string{"cpu-power", "instance-type"},
	},
	{
		// Ambiguous constraint errors take precedence over unsupported errors.
		cons:        "root-disk=8G mem=4G cpu-cores=4 instance-type=foo",
		reds:        []string{"mem", "arch"},
		blues:       []string{"instance-type"},
		unsupported: []string{"cpu-cores"},
		err:         `ambiguous constraints: "instance-type" overlaps with "mem"`,
	},
	{
		cons: "root-disk=8G mem=4G arch=amd64 cpu-cores=4 instance-type=foo",
		reds: []string{"mem", "arch"},
		err:  "",
	},
	{
		cons:  "root-disk=8G mem=4G arch=amd64 cpu-cores=4 instance-type=foo",
		blues: []string{"mem", "arch"},
		err:   "",
	},
	{
		cons:  "root-disk=8G mem=4G arch=amd64 cpu-cores=4 instance-type=foo",
		reds:  []string{"mem", "arch"},
		blues: []string{"instance-type"},
		err:   `ambiguous constraints: "arch" overlaps with "instance-type"`,
	},
	{
		cons:  "root-disk=8G mem=4G arch=amd64 cpu-cores=4 instance-type=foo",
		reds:  []string{"instance-type"},
		blues: []string{"mem", "arch"},
		err:   `ambiguous constraints: "arch" overlaps with "instance-type"`,
	},
	{
		cons:  "root-disk=8G mem=4G cpu-cores=4 instance-type=foo",
		reds:  []string{"mem", "arch"},
		blues: []string{"instance-type"},
		err:   `ambiguous constraints: "instance-type" overlaps with "mem"`,
	},
	{
		cons:  "arch=amd64 mem=4G cpu-cores=4",
		vocab: map[string][]interface{}{"arch": {"amd64", "i386"}},
	},
	{
		cons:  "mem=4G cpu-cores=4",
		vocab: map[string][]interface{}{"cpu-cores": {2, 4, 8}},
	},
	{
		cons:  "mem=4G instance-type=foo",
		vocab: map[string][]interface{}{"instance-type": {"foo", "bar"}},
	},
	{
		cons:  "mem=4G tags=foo,bar",
		vocab: map[string][]interface{}{"tags": {"foo", "bar", "another"}},
	},
	{
		cons:  "arch=i386 mem=4G cpu-cores=4",
		vocab: map[string][]interface{}{"arch": {"amd64"}},
		err:   "invalid constraint value: arch=i386\nvalid values are:.*",
	},
	{
		cons:  "mem=4G cpu-cores=5",
		vocab: map[string][]interface{}{"cpu-cores": {2, 4, 8}},
		err:   "invalid constraint value: cpu-cores=5\nvalid values are:.*",
	},
	{
		cons:  "mem=4G instance-type=foo",
		vocab: map[string][]interface{}{"instance-type": {"bar"}},
		err:   "invalid constraint value: instance-type=foo\nvalid values are:.*",
	},
	{
		cons:  "mem=4G tags=foo,other",
		vocab: map[string][]interface{}{"tags": {"foo", "bar", "another"}},
		err:   "invalid constraint value: tags=other\nvalid values are:.*",
	},
	{
		cons: "arch=i386 mem=4G instance-type=foo",
		vocab: map[string][]interface{}{
			"instance-type": {"foo", "bar"},
			"arch":          {"amd64", "i386"}},
	},
	{
		cons:  "virt-type=bar",
		vocab: map[string][]interface{}{"virt-type": {"bar"}},
	},
}

func (s *validationSuite) TestValidation(c *gc.C) {
	for i, t := range validationTests {
		c.Logf("test %d", i)
		validator := constraints.NewValidator()
		validator.RegisterUnsupported(t.unsupported)
		validator.RegisterConflicts(t.reds, t.blues)
		for a, v := range t.vocab {
			validator.RegisterVocabulary(a, v)
		}
		cons := constraints.MustParse(t.cons)
		unsupported, err := validator.Validate(cons)
		if t.err == "" {
			c.Assert(err, jc.ErrorIsNil)
			c.Assert(unsupported, jc.SameContents, t.unsupported)
		} else {
			c.Assert(err, gc.ErrorMatches, t.err)
		}
	}
}

var mergeTests = []struct {
	desc         string
	consFallback string
	cons         string
	unsupported  []string
	reds         []string
	blues        []string
	expected     string
}{
	{
		desc: "empty all round",
	}, {
		desc:     "container with empty fallback",
		cons:     "container=lxd",
		expected: "container=lxd",
	}, {
		desc:         "container from fallback",
		consFallback: "container=lxd",
		expected:     "container=lxd",
	}, {
		desc:     "arch with empty fallback",
		cons:     "arch=amd64",
		expected: "arch=amd64",
	}, {
		desc:         "arch with ignored fallback",
		cons:         "arch=amd64",
		consFallback: "arch=i386",
		expected:     "arch=amd64",
	}, {
		desc:         "arch from fallback",
		consFallback: "arch=i386",
		expected:     "arch=i386",
	}, {
		desc:     "instance type with empty fallback",
		cons:     "instance-type=foo",
		expected: "instance-type=foo",
	}, {
		desc:         "instance type with ignored fallback",
		cons:         "instance-type=foo",
		consFallback: "instance-type=bar",
		expected:     "instance-type=foo",
	}, {
		desc:         "instance type from fallback",
		consFallback: "instance-type=foo",
		expected:     "instance-type=foo",
	}, {
		desc:     "cpu-cores with empty fallback",
		cons:     "cpu-cores=2",
		expected: "cpu-cores=2",
	}, {
		desc:         "cpu-cores with ignored fallback",
		cons:         "cpu-cores=4",
		consFallback: "cpu-cores=8",
		expected:     "cpu-cores=4",
	}, {
		desc:         "cpu-cores from fallback",
		consFallback: "cpu-cores=8",
		expected:     "cpu-cores=8",
	}, {
		desc:     "cpu-power with empty fallback",
		cons:     "cpu-power=100",
		expected: "cpu-power=100",
	}, {
		desc:         "cpu-power with ignored fallback",
		cons:         "cpu-power=100",
		consFallback: "cpu-power=200",
		expected:     "cpu-power=100",
	}, {
		desc:         "cpu-power from fallback",
		consFallback: "cpu-power=200",
		expected:     "cpu-power=200",
	}, {
		desc:     "tags with empty fallback",
		cons:     "tags=foo,bar",
		expected: "tags=foo,bar",
	}, {
		desc:         "tags with ignored fallback",
		cons:         "tags=foo,bar",
		consFallback: "tags=baz",
		expected:     "tags=foo,bar",
	}, {
		desc:         "tags from fallback",
		consFallback: "tags=foo,bar",
		expected:     "tags=foo,bar",
	}, {
		desc:         "tags inital empty",
		cons:         "tags=",
		consFallback: "tags=foo,bar",
		expected:     "tags=",
	}, {
		desc:     "mem with empty fallback",
		cons:     "mem=4G",
		expected: "mem=4G",
	}, {
		desc:         "mem with ignored fallback",
		cons:         "mem=4G",
		consFallback: "mem=8G",
		expected:     "mem=4G",
	}, {
		desc:         "mem from fallback",
		consFallback: "mem=8G",
		expected:     "mem=8G",
	}, {
		desc:     "root-disk with empty fallback",
		cons:     "root-disk=4G",
		expected: "root-disk=4G",
	}, {
		desc:         "root-disk with ignored fallback",
		cons:         "root-disk=4G",
		consFallback: "root-disk=8G",
		expected:     "root-disk=4G",
	}, {
		desc:         "root-disk from fallback",
		consFallback: "root-disk=8G",
		expected:     "root-disk=8G",
	}, {
		desc:         "non-overlapping mix",
		cons:         "root-disk=8G mem=4G arch=amd64",
		consFallback: "cpu-power=1000 cpu-cores=4",
		expected:     "root-disk=8G mem=4G arch=amd64 cpu-power=1000 cpu-cores=4",
	}, {
		desc:         "overlapping mix",
		cons:         "root-disk=8G mem=4G arch=amd64",
		consFallback: "cpu-power=1000 cpu-cores=4 mem=8G",
		expected:     "root-disk=8G mem=4G arch=amd64 cpu-power=1000 cpu-cores=4",
	}, {
		desc:         "fallback only, no conflicts",
		consFallback: "root-disk=8G cpu-cores=4 instance-type=foo",
		reds:         []string{"mem", "arch"},
		blues:        []string{"instance-type"},
		expected:     "root-disk=8G cpu-cores=4 instance-type=foo",
	}, {
		desc:     "no fallback, no conflicts",
		cons:     "root-disk=8G cpu-cores=4 instance-type=foo",
		reds:     []string{"mem", "arch"},
		blues:    []string{"instance-type"},
		expected: "root-disk=8G cpu-cores=4 instance-type=foo",
	}, {
		desc:         "conflict value from override",
		consFallback: "root-disk=8G instance-type=foo",
		cons:         "root-disk=8G cpu-cores=4 instance-type=bar",
		reds:         []string{"mem", "arch"},
		blues:        []string{"instance-type"},
		expected:     "root-disk=8G cpu-cores=4 instance-type=bar",
	}, {
		desc:         "unsupported attributes ignored",
		consFallback: "root-disk=8G instance-type=foo",
		cons:         "root-disk=8G cpu-cores=4 instance-type=bar",
		reds:         []string{"mem", "arch"},
		blues:        []string{"instance-type"},
		unsupported:  []string{"instance-type"},
		expected:     "root-disk=8G cpu-cores=4 instance-type=bar",
	}, {
		desc:         "red conflict masked from fallback",
		consFallback: "root-disk=8G mem=4G",
		cons:         "root-disk=8G cpu-cores=4 instance-type=bar",
		reds:         []string{"mem", "arch"},
		blues:        []string{"instance-type"},
		expected:     "root-disk=8G cpu-cores=4 instance-type=bar",
	}, {
		desc:         "second red conflict masked from fallback",
		consFallback: "root-disk=8G arch=amd64",
		cons:         "root-disk=8G cpu-cores=4 instance-type=bar",
		reds:         []string{"mem", "arch"},
		blues:        []string{"instance-type"},
		expected:     "root-disk=8G cpu-cores=4 instance-type=bar",
	}, {
		desc:         "blue conflict masked from fallback",
		consFallback: "root-disk=8G cpu-cores=4 instance-type=bar",
		cons:         "root-disk=8G mem=4G",
		reds:         []string{"mem", "arch"},
		blues:        []string{"instance-type"},
		expected:     "root-disk=8G cpu-cores=4 mem=4G",
	}, {
		desc:         "both red conflicts used, blue mased from fallback",
		consFallback: "root-disk=8G cpu-cores=4 instance-type=bar",
		cons:         "root-disk=8G arch=amd64 mem=4G",
		reds:         []string{"mem", "arch"},
		blues:        []string{"instance-type"},
		expected:     "root-disk=8G cpu-cores=4 arch=amd64 mem=4G",
	},
}

func (s *validationSuite) TestMerge(c *gc.C) {
	for i, t := range mergeTests {
		c.Logf("test %d: %s", i, t.desc)
		validator := constraints.NewValidator()
		validator.RegisterConflicts(t.reds, t.blues)
		consFallback := constraints.MustParse(t.consFallback)
		cons := constraints.MustParse(t.cons)
		merged, err := validator.Merge(consFallback, cons)
		c.Assert(err, jc.ErrorIsNil)
		expected := constraints.MustParse(t.expected)
		c.Check(merged, gc.DeepEquals, expected)
	}
}

func (s *validationSuite) TestMergeError(c *gc.C) {
	validator := constraints.NewValidator()
	validator.RegisterConflicts([]string{"instance-type"}, []string{"mem"})
	consFallback := constraints.MustParse("instance-type=foo mem=4G")
	cons := constraints.MustParse("cpu-cores=2")
	_, err := validator.Merge(consFallback, cons)
	c.Assert(err, gc.ErrorMatches, `ambiguous constraints: "instance-type" overlaps with "mem"`)
	_, err = validator.Merge(cons, consFallback)
	c.Assert(err, gc.ErrorMatches, `ambiguous constraints: "instance-type" overlaps with "mem"`)
}

func (s *validationSuite) TestUpdateVocabulary(c *gc.C) {
	validator := constraints.NewValidator()
	attributeName := "arch"
	originalValues := []string{"amd64"}
	validator.RegisterVocabulary(attributeName, originalValues)

	cons := constraints.MustParse("arch=amd64")
	_, err := validator.Validate(cons)
	c.Assert(err, jc.ErrorIsNil)

	cons2 := constraints.MustParse("arch=ppc64el")
	_, err = validator.Validate(cons2)
	c.Assert(err, gc.ErrorMatches, regexp.QuoteMeta(`invalid constraint value: arch=ppc64el
valid values are: [amd64]`))

	additionalValues := []string{"ppc64el"}
	validator.UpdateVocabulary(attributeName, additionalValues)

	_, err = validator.Validate(cons)
	c.Assert(err, jc.ErrorIsNil)
	_, err = validator.Validate(cons2)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *intersectionSuite) SetUpTest(c *gc.C) {
	s.validCons = "arch=amd64"
	s.invalidCons = "arch=ppc64el"
}

func (s *intersectionSuite) TearDownTest(c *gc.C) {
	s.originalAttributeValues = nil
	s.additionalAttributeValues = nil
	s.errorBeforeIntersection = ""
	s.errorForValidConsAfterIntersection = ""
	s.errorAfterIntersection = ""
}

func (s *intersectionSuite) TestIntersectVocabularyNone(c *gc.C) {
	s.originalAttributeValues = []string{"amd64"}
	s.additionalAttributeValues = []string{"ppc64el"}
	s.errorBeforeIntersection = `invalid constraint value: arch=ppc64el
valid values are: [amd64]`
	s.errorForValidConsAfterIntersection = `invalid constraint value: arch=amd64
valid values are: []`
	s.errorAfterIntersection = `invalid constraint value: arch=ppc64el
valid values are: []`
	s.assertVocabularyValuesIntersected(c)
}

func (s *intersectionSuite) TestIntersectVocabularyNewValueSetEmpty(c *gc.C) {
	s.originalAttributeValues = []string{"amd64"}
	s.additionalAttributeValues = []string{}
	s.errorBeforeIntersection = `invalid constraint value: arch=ppc64el
valid values are: [amd64]`
	s.errorForValidConsAfterIntersection = `invalid constraint value: arch=amd64
valid values are: []`
	s.errorAfterIntersection = `invalid constraint value: arch=ppc64el
valid values are: []`
	s.assertVocabularyValuesIntersected(c)
}

func (s *intersectionSuite) TestIntersectVocabularyOldValueSetEmpty(c *gc.C) {
	s.additionalAttributeValues = []string{"ppc64el"}
	s.errorForValidConsAfterIntersection = `invalid constraint value: arch=amd64
valid values are: []`
	s.errorAfterIntersection = `invalid constraint value: arch=ppc64el
valid values are: []`
	s.assertVocabularyValuesIntersected(c)
}

func (s *intersectionSuite) TestIntersectVocabularyBothValueSetsEmpty(c *gc.C) {
	s.additionalAttributeValues = []string{}
	s.assertVocabularyValuesIntersected(c)
}

func (s *intersectionSuite) TestIntersectVocabularyOneValue(c *gc.C) {
	s.originalAttributeValues = []string{"amd64", "s390"}
	s.additionalAttributeValues = []string{"ppc64el", "amd64"}
	s.errorBeforeIntersection = `invalid constraint value: arch=ppc64el
valid values are: [amd64 s390]`
	s.errorAfterIntersection = `invalid constraint value: arch=ppc64el
valid values are: [amd64]`
	s.assertVocabularyValuesIntersected(c)
}

func (s *intersectionSuite) TestIntersectVocabularyMultipleValues(c *gc.C) {
	s.originalAttributeValues = []string{"amd64", "s390"}
	s.additionalAttributeValues = []string{"ppc64el", "amd64", "s390"}
	s.errorBeforeIntersection = `invalid constraint value: arch=ppc64el
valid values are: [amd64 s390]`
	s.errorAfterIntersection = `invalid constraint value: arch=ppc64el
valid values are: [amd64 s390]`
	s.assertVocabularyValuesIntersected(c)
}

func (s *intersectionSuite) TestIntersectVocabularyOldValueSetLonger(c *gc.C) {
	s.originalAttributeValues = []string{"amd64", "s390"}
	s.additionalAttributeValues = []string{"amd64"}
	s.errorBeforeIntersection = `invalid constraint value: arch=ppc64el
valid values are: [amd64 s390]`
	s.errorAfterIntersection = `invalid constraint value: arch=ppc64el
valid values are: [amd64]`
	s.assertVocabularyValuesIntersected(c)
}

func (s *intersectionSuite) assertVocabularyValuesIntersected(c *gc.C) {
	validator := constraints.NewValidator()
	attributeName := "arch"
	if s.originalAttributeValues != nil {
		validator.RegisterVocabulary(attributeName, s.originalAttributeValues)
	}

	validCons := constraints.MustParse(s.validCons)
	_, err := validator.Validate(validCons)
	c.Assert(err, jc.ErrorIsNil)

	invalidCons := constraints.MustParse(s.invalidCons)
	_, err = validator.Validate(invalidCons)
	if s.errorBeforeIntersection != "" {
		c.Assert(err, gc.ErrorMatches, regexp.QuoteMeta(s.errorBeforeIntersection))
	} else {
		c.Assert(err, jc.ErrorIsNil)
	}

	validator.IntersectVocabulary(attributeName, s.additionalAttributeValues)

	_, err = validator.Validate(validCons)
	if s.errorForValidConsAfterIntersection != "" {
		c.Assert(err, gc.ErrorMatches, regexp.QuoteMeta(s.errorForValidConsAfterIntersection))
	} else {
		c.Assert(err, jc.ErrorIsNil)
	}
	_, err = validator.Validate(invalidCons)
	if s.errorAfterIntersection != "" {
		c.Assert(err, gc.ErrorMatches, regexp.QuoteMeta(s.errorAfterIntersection))
	} else {
		c.Assert(err, jc.ErrorIsNil)
	}
}

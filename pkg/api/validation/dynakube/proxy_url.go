package validation

import (
	"context"
	"net/url"
	"regexp"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/pkg/errors"
)

const (
	errorInvalidProxyUrl      = `The DynaKube's specification has an invalid Proxy URL value set. Make sure you correctly specify the URL in your custom resource or in the provided secret.`
	errorInvalidEvalCharacter = `The DynaKube's specification has an invalid Proxy password value set. Make sure you don't use forbidden characters: space, apostrophe, backtick, comma, ampersand, equals sign, plus sign, percent sign, backslash.`

	errorMissingProxySecret = `Error occurred while reading PROXY secret indicated in the Dynakube specification`
)

func invalidActiveGateProxyUrl(ctx context.Context, dv *Validator, dk *dynakube.DynaKube) string {
	if dk.Spec.Proxy != nil {
		proxyUrl, err := dk.Proxy(ctx, dv.apiReader)
		if err != nil {
			return errors.Wrap(err, errorMissingProxySecret).Error()
		}

		return validateProxyUrl(proxyUrl, errorInvalidProxyUrl, errorInvalidEvalCharacter)
	}

	return ""
}

// proxyUrl is valid if
// 1) encoded
// 2) password does not contain '` characters.
func validateProxyUrl(proxyUrl string, parseErrorMessage string, evalErrorMessage string) string {
	if parsedUrl, err := url.Parse(proxyUrl); err != nil {
		return parseErrorMessage
	} else {
		password, _ := parsedUrl.User.Password()
		if !isStringValidForAG(password) {
			return evalErrorMessage
		}
	}

	return ""
}

func isStringValidForAG(str string) bool {
	// SP   !	"	#	$	%	&	'	(	)	*	+	,	-	.	/
	// 0	1	2	3	4	5	6	7	8	9	:	;	<	=	>	?
	// @	A	B	C	D	E	F	G	H	I	J	K	L	M	N	O
	// P	Q	R	S	T	U	V	W	X	Y	Z	[	\	]	^	_
	// `	a	b	c	d	e	f	g	h	i	j	k	l	m	n	o
	// p	q	r	s	t	u	v	w	x	y	z	{	|	}	~
	// '\'' '`'            exceptions due to entrypoint.sh:readSecret:eval
	// ','                 exceptions due to Gateway reader of config files
	// '&' '=' '+' '%' '\' exceptions due to entrypoint.sh:saveProxyConfiguration
	regex := regexp.MustCompile(`^[!"#$()*\-./0-9:;<>?@A-Z\[\]^_a-z{|}~]*$`)

	return regex.MatchString(str)
}

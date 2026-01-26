package validation

import (
	"context"
	"net/url"
	"regexp"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/pkg/errors"
)

const (
	errorInvalidProxyURL      = `The DynaKube's specification has an invalid Proxy URL value set. Make sure you correctly specify the URL in your custom resource or in the provided secret.`
	errorInvalidEvalCharacter = `The DynaKube's specification has an invalid Proxy password value set. Make sure you don't use forbidden characters: space, apostrophe, backtick, comma, ampersand, equals sign, plus sign, percent sign, backslash.`

	errorMissingProxySecret = `Error occurred while reading PROXY secret indicated in the Dynakube specification`
)

func invalidActiveGateProxyURL(ctx context.Context, dv *validatorClient, dk *dynakube.DynaKube) string {
	if dk.Spec.Proxy != nil {
		proxyURL, err := dk.Proxy(ctx, dv.apiReader)
		if err != nil {
			return errors.Wrap(err, errorMissingProxySecret).Error()
		}

		return validateProxyURL(proxyURL, errorInvalidProxyURL, errorInvalidEvalCharacter)
	}

	return ""
}

// proxyURL is valid if
// 1) encoded
// 2) password does not contain '` characters.
func validateProxyURL(proxyURL string, parseErrorMessage string, evalErrorMessage string) string {
	if parsedURL, err := url.Parse(proxyURL); err != nil {
		return parseErrorMessage
	} else {
		password, _ := parsedURL.User.Password()
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

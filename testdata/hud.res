#base "base.res"

"Resource/UI/Hud.res"
{
	"Label"
	{
		"labelText"	"Line one\nLine two"
		"tooltip"	"Say \"hi\""
		"wide"		"200"	[$WIN32]
		"wide"		"100"	[$X360]
	}
	"Panel" [!$OSX]
	{
		"visible"	"1"
	}
}

// Code generated by "stringer -type=SymKind symkind.go"; DO NOT EDIT.

package sym

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[Sxxx-0]
	_ = x[STEXT-1]
	_ = x[SELFRXSECT-2]
	_ = x[SMACHOPLT-3]
	_ = x[STYPE-4]
	_ = x[SSTRING-5]
	_ = x[SGOSTRING-6]
	_ = x[SGOFUNC-7]
	_ = x[SGCBITS-8]
	_ = x[SRODATA-9]
	_ = x[SFUNCTAB-10]
	_ = x[SELFROSECT-11]
	_ = x[STYPERELRO-12]
	_ = x[SSTRINGRELRO-13]
	_ = x[SGOSTRINGRELRO-14]
	_ = x[SGOFUNCRELRO-15]
	_ = x[SGCBITSRELRO-16]
	_ = x[SRODATARELRO-17]
	_ = x[SFUNCTABRELRO-18]
	_ = x[STYPELINK-19]
	_ = x[SITABLINK-20]
	_ = x[SSYMTAB-21]
	_ = x[SPCLNTAB-22]
	_ = x[SFirstWritable-23]
	_ = x[SBUILDINFO-24]
	_ = x[SELFSECT-25]
	_ = x[SMACHO-26]
	_ = x[SMACHOGOT-27]
	_ = x[SWINDOWS-28]
	_ = x[SELFGOT-29]
	_ = x[SNOPTRDATA-30]
	_ = x[SINITARR-31]
	_ = x[SDATA-32]
	_ = x[SXCOFFTOC-33]
	_ = x[SBSS-34]
	_ = x[SNOPTRBSS-35]
	_ = x[SLIBFUZZER_8BIT_COUNTER-36]
	_ = x[SCOVERAGE_COUNTER-37]
	_ = x[SCOVERAGE_AUXVAR-38]
	_ = x[STLSBSS-39]
	_ = x[SXREF-40]
	_ = x[SMACHOSYMSTR-41]
	_ = x[SMACHOSYMTAB-42]
	_ = x[SMACHOINDIRECTPLT-43]
	_ = x[SMACHOINDIRECTGOT-44]
	_ = x[SFILEPATH-45]
	_ = x[SDYNIMPORT-46]
	_ = x[SHOSTOBJ-47]
	_ = x[SUNDEFEXT-48]
	_ = x[SDWARFSECT-49]
	_ = x[SDWARFCUINFO-50]
	_ = x[SDWARFCONST-51]
	_ = x[SDWARFFCN-52]
	_ = x[SDWARFABSFCN-53]
	_ = x[SDWARFTYPE-54]
	_ = x[SDWARFVAR-55]
	_ = x[SDWARFRANGE-56]
	_ = x[SDWARFLOC-57]
	_ = x[SDWARFLINES-58]
}

const _SymKind_name = "SxxxSTEXTSELFRXSECTSMACHOPLTSTYPESSTRINGSGOSTRINGSGOFUNCSGCBITSSRODATASFUNCTABSELFROSECTSTYPERELROSSTRINGRELROSGOSTRINGRELROSGOFUNCRELROSGCBITSRELROSRODATARELROSFUNCTABRELROSTYPELINKSITABLINKSSYMTABSPCLNTABSFirstWritableSBUILDINFOSELFSECTSMACHOSMACHOGOTSWINDOWSSELFGOTSNOPTRDATASINITARRSDATASXCOFFTOCSBSSSNOPTRBSSSLIBFUZZER_8BIT_COUNTERSCOVERAGE_COUNTERSCOVERAGE_AUXVARSTLSBSSSXREFSMACHOSYMSTRSMACHOSYMTABSMACHOINDIRECTPLTSMACHOINDIRECTGOTSFILEPATHSDYNIMPORTSHOSTOBJSUNDEFEXTSDWARFSECTSDWARFCUINFOSDWARFCONSTSDWARFFCNSDWARFABSFCNSDWARFTYPESDWARFVARSDWARFRANGESDWARFLOCSDWARFLINES"

var _SymKind_index = [...]uint16{0, 4, 9, 19, 28, 33, 40, 49, 56, 63, 70, 78, 88, 98, 110, 124, 136, 148, 160, 173, 182, 191, 198, 206, 220, 230, 238, 244, 253, 261, 268, 278, 286, 291, 300, 304, 313, 336, 353, 369, 376, 381, 393, 405, 422, 439, 448, 458, 466, 475, 485, 497, 508, 517, 529, 539, 548, 559, 568, 579}

func (i SymKind) String() string {
	if i >= SymKind(len(_SymKind_index)-1) {
		return "SymKind(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _SymKind_name[_SymKind_index[i]:_SymKind_index[i+1]]
}

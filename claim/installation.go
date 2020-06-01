package claim

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

// Installation is a non-storage construct representing an installation of a
// bundle.
type Installation struct {
	Name string
	Claims
}

// NewInstallation creates an Installation and ensures the contained data is sorted.
func NewInstallation(name string, claims []Claim) Installation {
	i := Installation{
		Name:   name,
		Claims: claims,
	}

	sort.Sort(i.Claims)
	for _, c := range i.Claims {
		sort.Sort(c.results)
	}

	return i
}

// GetInstallationTimestamp searches the claims associated with the installation
// for the first claim for Install and returns its timestamp.
func (i Installation) GetInstallationTimestamp() (time.Time, error) {
	if len(i.Claims) == 0 {
		return time.Time{}, fmt.Errorf("the installation %s has no claims", i.Name)
	}

	for _, c := range i.Claims {
		if c.Action == ActionInstall {
			return c.Created, nil
		}
	}

	return time.Time{}, fmt.Errorf("the installation %s has never been installed", i.Name)
}

// GetLastClaim returns the most recent (last) claim associated with the
// installation.
func (i Installation) GetLastClaim() (Claim, error) {
	if len(i.Claims) == 0 {
		return Claim{}, fmt.Errorf("the installation %s has no claims", i.Name)
	}

	lastClaim := i.Claims[len(i.Claims)-1]
	return lastClaim, nil
}

// GetLastResult returns the most recent (last) result associated with the
// installation.
func (i Installation) GetLastResult() (Result, error) {
	lastClaim, err := i.GetLastClaim()
	if err != nil {
		return Result{}, err
	}

	if len(lastClaim.results) == 0 {
		return Result{}, errors.New("the last claim has no results")
	}

	lastResult := lastClaim.results[len(lastClaim.results)-1]
	return lastResult, nil
}

// GetLastStatus returns the status from the most recent (last) result
// associated with the installation, or "unknown" if it cannot be determined.
func (i Installation) GetLastStatus() string {
	lastResult, err := i.GetLastResult()
	if err != nil {
		return StatusUnknown
	}

	return lastResult.Status
}

type InstallationByName []Installation

func (ibn InstallationByName) Len() int {
	return len(ibn)
}

func (ibn InstallationByName) Less(i, j int) bool {
	return ibn[i].Name < ibn[j].Name
}

func (ibn InstallationByName) Swap(i, j int) {
	ibn[i], ibn[j] = ibn[j], ibn[i]
}

// InstallationByModified sorts installations by which has been modified most recently
// Assumes that the installation's claims have already been sorted first, for example
// with SortClaims or manually.
type InstallationByModified []Installation

// SortClaims presorts the claims on each installation before the
// installations can be sorted.
func (ibm InstallationByModified) SortClaims() {
	for _, i := range ibm {
		sort.Sort(i.Claims)
	}
}

func (ibm InstallationByModified) Len() int {
	return len(ibm)
}

func (ibm InstallationByModified) Less(i, j int) bool {
	ic := ibm[i].Claims[len(ibm[i].Claims)-1]
	jc := ibm[j].Claims[len(ibm[j].Claims)-1]

	return ic.ID < jc.ID
}

func (ibm InstallationByModified) Swap(i, j int) {
	ibm[i], ibm[j] = ibm[j], ibm[i]
}

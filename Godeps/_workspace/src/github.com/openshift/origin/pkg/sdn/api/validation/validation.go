package validation

import (
	"net"

	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/util/fielderrors"

	oapi "github.com/openshift/origin/pkg/api"
	sdnapi "github.com/openshift/origin/pkg/sdn/api"
)

// ValidateClusterNetwork tests if required fields in the ClusterNetwork are set.
func ValidateClusterNetwork(clusterNet *sdnapi.ClusterNetwork) fielderrors.ValidationErrorList {
	allErrs := fielderrors.ValidationErrorList{}
	allErrs = append(allErrs, validation.ValidateObjectMeta(&clusterNet.ObjectMeta, false, oapi.MinimalNameRequirements).Prefix("metadata")...)

	_, ipnet, err := net.ParseCIDR(clusterNet.Network)
	if err != nil {
		allErrs = append(allErrs, fielderrors.NewFieldInvalid("network", clusterNet.Network, err.Error()))
	} else {
		ones, bitSize := ipnet.Mask.Size()
		if (bitSize - ones) <= clusterNet.HostSubnetLength {
			allErrs = append(allErrs, fielderrors.NewFieldInvalid("hostSubnetLength", clusterNet.HostSubnetLength, "subnet length is greater than cluster Mask"))
		}
	}

	return allErrs
}

func ValidateClusterNetworkUpdate(obj *sdnapi.ClusterNetwork, old *sdnapi.ClusterNetwork) fielderrors.ValidationErrorList {
	allErrs := fielderrors.ValidationErrorList{}
	allErrs = append(allErrs, validation.ValidateObjectMetaUpdate(&obj.ObjectMeta, &old.ObjectMeta).Prefix("metadata")...)

	if obj.Network != old.Network {
		allErrs = append(allErrs, fielderrors.NewFieldInvalid("Network", obj.Network, "cannot change the cluster's network CIDR midflight."))
	}

	return allErrs
}

// ValidateHostSubnet tests fields for the host subnet, the host should be a network resolvable string,
//  and subnet should be a valid CIDR
func ValidateHostSubnet(hs *sdnapi.HostSubnet) fielderrors.ValidationErrorList {
	allErrs := fielderrors.ValidationErrorList{}
	allErrs = append(allErrs, validation.ValidateObjectMeta(&hs.ObjectMeta, false, oapi.MinimalNameRequirements).Prefix("metadata")...)

	_, _, err := net.ParseCIDR(hs.Subnet)
	if err != nil {
		allErrs = append(allErrs, fielderrors.NewFieldInvalid("subnet", hs.Subnet, err.Error()))
	}
	if net.ParseIP(hs.HostIP) == nil {
		allErrs = append(allErrs, fielderrors.NewFieldInvalid("hostIP", hs.HostIP, "invalid IP address"))
	}
	return allErrs
}

func ValidateHostSubnetUpdate(obj *sdnapi.HostSubnet, old *sdnapi.HostSubnet) fielderrors.ValidationErrorList {
	allErrs := fielderrors.ValidationErrorList{}
	allErrs = append(allErrs, validation.ValidateObjectMetaUpdate(&obj.ObjectMeta, &old.ObjectMeta).Prefix("metadata")...)

	if obj.Subnet != old.Subnet {
		allErrs = append(allErrs, fielderrors.NewFieldInvalid("subnet", obj.Subnet, "cannot change the subnet lease midflight."))
	}

	return allErrs
}

// ValidateNetNamespace tests fields for a greater-than-zero NetID
func ValidateNetNamespace(netnamespace *sdnapi.NetNamespace) fielderrors.ValidationErrorList {
	allErrs := fielderrors.ValidationErrorList{}
	allErrs = append(allErrs, validation.ValidateObjectMeta(&netnamespace.ObjectMeta, false, oapi.MinimalNameRequirements).Prefix("metadata")...)

	if netnamespace.NetID < 0 {
		allErrs = append(allErrs, fielderrors.NewFieldInvalid("netID", netnamespace.NetID, "invalid Net ID: cannot be negative"))
	}
	return allErrs
}

func ValidateNetNamespaceUpdate(obj *sdnapi.NetNamespace, old *sdnapi.NetNamespace) fielderrors.ValidationErrorList {
	allErrs := fielderrors.ValidationErrorList{}
	allErrs = append(allErrs, validation.ValidateObjectMetaUpdate(&obj.ObjectMeta, &old.ObjectMeta).Prefix("metadata")...)

	if obj.NetID != old.NetID {
		allErrs = append(allErrs, fielderrors.NewFieldInvalid("netid", obj.NetID, "cannot change the NetID midflight."))
	}

	return allErrs
}

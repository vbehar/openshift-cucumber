package v1

import (
	"fmt"
	"math"
	"strings"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/conversion"
	kutil "k8s.io/kubernetes/pkg/util"

	oapi "github.com/openshift/origin/pkg/api"
	newer "github.com/openshift/origin/pkg/deploy/api"
	imageapi "github.com/openshift/origin/pkg/image/api"
)

func convert_v1_DeploymentTriggerImageChangeParams_To_api_DeploymentTriggerImageChangeParams(in *DeploymentTriggerImageChangeParams, out *newer.DeploymentTriggerImageChangeParams, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	switch in.From.Kind {
	case "ImageStreamTag":
	case "ImageStream", "ImageRepository":
		out.From.Kind = "ImageStreamTag"
		if !strings.Contains(out.From.Name, ":") {
			out.From.Name = imageapi.JoinImageStreamTag(out.From.Name, imageapi.DefaultImageTag)
		}
	default:
		// Will be handled by validation
	}
	return nil
}

func convert_api_DeploymentTriggerImageChangeParams_To_v1_DeploymentTriggerImageChangeParams(in *newer.DeploymentTriggerImageChangeParams, out *DeploymentTriggerImageChangeParams, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	switch in.From.Kind {
	case "ImageStreamTag":
	case "ImageStream", "ImageRepository":
		out.From.Kind = "ImageStreamTag"
		if !strings.Contains(out.From.Name, ":") {
			out.From.Name = imageapi.JoinImageStreamTag(out.From.Name, imageapi.DefaultImageTag)
		}
	default:
		// Will be handled by validation
	}
	return nil
}

func convert_v1_RollingDeploymentStrategyParams_To_api_RollingDeploymentStrategyParams(in *RollingDeploymentStrategyParams, out *newer.RollingDeploymentStrategyParams, s conversion.Scope) error {
	out.UpdatePeriodSeconds = in.UpdatePeriodSeconds
	out.IntervalSeconds = in.IntervalSeconds
	out.TimeoutSeconds = in.TimeoutSeconds
	out.UpdatePercent = in.UpdatePercent

	if in.Pre != nil {
		if err := s.Convert(&in.Pre, &out.Pre, 0); err != nil {
			return err
		}
	}
	if in.Post != nil {
		if err := s.Convert(&in.Post, &out.Post, 0); err != nil {
			return err
		}
	}

	if in.UpdatePercent != nil {
		pct := kutil.NewIntOrStringFromString(fmt.Sprintf("%d%%", int(math.Abs(float64(*in.UpdatePercent)))))
		if *in.UpdatePercent > 0 {
			out.MaxSurge = pct
		} else {
			out.MaxUnavailable = pct
		}
	} else {
		if err := s.Convert(in.MaxUnavailable, &out.MaxUnavailable, 0); err != nil {
			return err
		}
		if err := s.Convert(in.MaxSurge, &out.MaxSurge, 0); err != nil {
			return err
		}
	}
	return nil
}

func convert_api_RollingDeploymentStrategyParams_To_v1_RollingDeploymentStrategyParams(in *newer.RollingDeploymentStrategyParams, out *RollingDeploymentStrategyParams, s conversion.Scope) error {
	out.UpdatePeriodSeconds = in.UpdatePeriodSeconds
	out.IntervalSeconds = in.IntervalSeconds
	out.TimeoutSeconds = in.TimeoutSeconds
	out.UpdatePercent = in.UpdatePercent

	if in.Pre != nil {
		if err := s.Convert(&in.Pre, &out.Pre, 0); err != nil {
			return err
		}
	}
	if in.Post != nil {
		if err := s.Convert(&in.Post, &out.Post, 0); err != nil {
			return err
		}
	}

	if out.MaxUnavailable == nil {
		out.MaxUnavailable = &kutil.IntOrString{}
	}
	if out.MaxSurge == nil {
		out.MaxSurge = &kutil.IntOrString{}
	}
	if in.UpdatePercent != nil {
		pct := kutil.NewIntOrStringFromString(fmt.Sprintf("%d%%", int(math.Abs(float64(*in.UpdatePercent)))))
		if *in.UpdatePercent > 0 {
			out.MaxSurge = &pct
		} else {
			out.MaxUnavailable = &pct
		}
	} else {
		if err := s.Convert(&in.MaxUnavailable, out.MaxUnavailable, 0); err != nil {
			return err
		}
		if err := s.Convert(&in.MaxSurge, out.MaxSurge, 0); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	err := api.Scheme.AddConversionFuncs(
		convert_v1_DeploymentTriggerImageChangeParams_To_api_DeploymentTriggerImageChangeParams,
		convert_api_DeploymentTriggerImageChangeParams_To_v1_DeploymentTriggerImageChangeParams,

		convert_v1_RollingDeploymentStrategyParams_To_api_RollingDeploymentStrategyParams,
		convert_api_RollingDeploymentStrategyParams_To_v1_RollingDeploymentStrategyParams,
	)
	if err != nil {
		panic(err)
	}

	if err := api.Scheme.AddFieldLabelConversionFunc("v1", "DeploymentConfig",
		oapi.GetFieldLabelConversionFunc(newer.DeploymentConfigToSelectableFields(&newer.DeploymentConfig{}), nil),
	); err != nil {
		panic(err)
	}
}

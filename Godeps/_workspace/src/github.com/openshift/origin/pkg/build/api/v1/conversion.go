package v1

import (
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/conversion"

	oapi "github.com/openshift/origin/pkg/api"
	newer "github.com/openshift/origin/pkg/build/api"
	imageapi "github.com/openshift/origin/pkg/image/api"
)

func convert_v1_SourceBuildStrategy_To_api_SourceBuildStrategy(in *SourceBuildStrategy, out *newer.SourceBuildStrategy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	switch in.From.Kind {
	case "ImageStream":
		out.From.Kind = "ImageStreamTag"
		out.From.Name = imageapi.JoinImageStreamTag(in.From.Name, "")
	}
	return nil
}

// empty conversion needed because the conversion generator can't handle unidirectional custom conversions
func convert_api_SourceBuildStrategy_To_v1_SourceBuildStrategy(in *newer.SourceBuildStrategy, out *SourceBuildStrategy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	return nil
}

func convert_v1_DockerBuildStrategy_To_api_DockerBuildStrategy(in *DockerBuildStrategy, out *newer.DockerBuildStrategy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	if in.From != nil {
		switch in.From.Kind {
		case "ImageStream":
			out.From.Kind = "ImageStreamTag"
			out.From.Name = imageapi.JoinImageStreamTag(in.From.Name, "")
		}
	}
	return nil
}

// empty conversion needed because the conversion generator can't handle unidirectional custom conversions
func convert_api_DockerBuildStrategy_To_v1_DockerBuildStrategy(in *newer.DockerBuildStrategy, out *DockerBuildStrategy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	return nil
}

func convert_v1_CustomBuildStrategy_To_api_CustomBuildStrategy(in *CustomBuildStrategy, out *newer.CustomBuildStrategy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	switch in.From.Kind {
	case "ImageStream":
		out.From.Kind = "ImageStreamTag"
		out.From.Name = imageapi.JoinImageStreamTag(in.From.Name, "")
	}
	return nil
}

// empty conversion needed because the conversion generator can't handle unidirectional custom conversions
func convert_api_CustomBuildStrategy_To_v1_CustomBuildStrategy(in *newer.CustomBuildStrategy, out *CustomBuildStrategy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	return nil
}

func convert_v1_BuildOutput_To_api_BuildOutput(in *BuildOutput, out *newer.BuildOutput, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	if in.To != nil && (in.To.Kind == "ImageStream" || len(in.To.Kind) == 0) {
		out.To.Kind = "ImageStreamTag"
		out.To.Name = imageapi.JoinImageStreamTag(in.To.Name, "")
	}
	return nil
}

// empty conversion needed because the conversion generator can't handle unidirectional custom conversions
func convert_api_BuildOutput_To_v1_BuildOutput(in *newer.BuildOutput, out *BuildOutput, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	return nil
}

// empty conversion needed because the conversion generator can't handle unidirectional custom conversions
func convert_api_BuildTriggerPolicy_To_v1_BuildTriggerPolicy(in *newer.BuildTriggerPolicy, out *BuildTriggerPolicy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.DestFromSource); err != nil {
		return err
	}
	return nil
}

func convert_v1_BuildTriggerPolicy_To_api_BuildTriggerPolicy(in *BuildTriggerPolicy, out *newer.BuildTriggerPolicy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.DestFromSource); err != nil {
		return err
	}
	switch in.Type {
	case ImageChangeBuildTriggerTypeDeprecated:
		out.Type = newer.ImageChangeBuildTriggerType
	case GenericWebHookBuildTriggerTypeDeprecated:
		out.Type = newer.GenericWebHookBuildTriggerType
	case GitHubWebHookBuildTriggerTypeDeprecated:
		out.Type = newer.GitHubWebHookBuildTriggerType
	}
	return nil
}

func convert_api_SourceRevision_To_v1_SourceRevision(in *newer.SourceRevision, out *SourceRevision, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	out.Type = BuildSourceGit
	return nil
}

func convert_v1_SourceRevision_To_api_SourceRevision(in *SourceRevision, out *newer.SourceRevision, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	return nil
}

func convert_api_BuildSource_To_v1_BuildSource(in *newer.BuildSource, out *BuildSource, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	switch {
	// it is legal for a buildsource to have both a git+dockerfile source, but in v1 that was represented
	// as type git.
	case in.Git != nil:
		out.Type = BuildSourceGit
	// it is legal for a buildsource to have both a binary+dockerfile source, but in v1 that was represented
	// as type binary.
	case in.Binary != nil:
		out.Type = BuildSourceBinary
	case in.Dockerfile != nil:
		out.Type = BuildSourceDockerfile
	}
	return nil
}

func convert_v1_BuildSource_To_api_BuildSource(in *BuildSource, out *newer.BuildSource, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	return nil
}

func convert_api_BuildStrategy_To_v1_BuildStrategy(in *newer.BuildStrategy, out *BuildStrategy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	switch {
	case in.SourceStrategy != nil:
		out.Type = SourceBuildStrategyType
	case in.DockerStrategy != nil:
		out.Type = DockerBuildStrategyType
	case in.CustomStrategy != nil:
		out.Type = CustomBuildStrategyType
	}
	return nil
}

func convert_v1_BuildStrategy_To_api_BuildStrategy(in *BuildStrategy, out *newer.BuildStrategy, s conversion.Scope) error {
	if err := s.DefaultConvert(in, out, conversion.IgnoreMissingFields); err != nil {
		return err
	}
	return nil
}

func init() {
	err := kapi.Scheme.AddDefaultingFuncs(
		func(strategy *BuildStrategy) {
			if (strategy != nil) && (strategy.Type == DockerBuildStrategyType) {
				//  initialize DockerStrategy to a default state if it's not set.
				if strategy.DockerStrategy == nil {
					strategy.DockerStrategy = &DockerBuildStrategy{}
				}
			}
		},
		func(obj *SourceBuildStrategy) {
			if len(obj.From.Kind) == 0 {
				obj.From.Kind = "ImageStreamTag"
			}
		},
		func(obj *DockerBuildStrategy) {
			if obj.From != nil && len(obj.From.Kind) == 0 {
				obj.From.Kind = "ImageStreamTag"
			}
		},
		func(obj *CustomBuildStrategy) {
			if len(obj.From.Kind) == 0 {
				obj.From.Kind = "ImageStreamTag"
			}
		},
		func(obj *BuildTriggerPolicy) {
			if obj.Type == ImageChangeBuildTriggerType && obj.ImageChange == nil {
				obj.ImageChange = &ImageChangeTrigger{}
			}
		},
	)
	if err != nil {
		panic(err)
	}

	kapi.Scheme.AddConversionFuncs(
		convert_v1_SourceBuildStrategy_To_api_SourceBuildStrategy,
		convert_api_SourceBuildStrategy_To_v1_SourceBuildStrategy,
		convert_v1_DockerBuildStrategy_To_api_DockerBuildStrategy,
		convert_api_DockerBuildStrategy_To_v1_DockerBuildStrategy,
		convert_v1_CustomBuildStrategy_To_api_CustomBuildStrategy,
		convert_api_CustomBuildStrategy_To_v1_CustomBuildStrategy,
		convert_v1_BuildOutput_To_api_BuildOutput,
		convert_api_BuildOutput_To_v1_BuildOutput,
		convert_v1_BuildTriggerPolicy_To_api_BuildTriggerPolicy,
		convert_api_BuildTriggerPolicy_To_v1_BuildTriggerPolicy,
		convert_v1_SourceRevision_To_api_SourceRevision,
		convert_api_SourceRevision_To_v1_SourceRevision,
		convert_v1_BuildSource_To_api_BuildSource,
		convert_api_BuildSource_To_v1_BuildSource,
		convert_v1_BuildStrategy_To_api_BuildStrategy,
		convert_api_BuildStrategy_To_v1_BuildStrategy,
	)

	if err := kapi.Scheme.AddFieldLabelConversionFunc("v1", "Build",
		oapi.GetFieldLabelConversionFunc(newer.BuildToSelectableFields(&newer.Build{}), map[string]string{"name": "metadata.name"}),
	); err != nil {
		panic(err)
	}

	if err := kapi.Scheme.AddFieldLabelConversionFunc("v1", "BuildConfig",
		oapi.GetFieldLabelConversionFunc(newer.BuildConfigToSelectableFields(&newer.BuildConfig{}), map[string]string{"name": "metadata.name"}),
	); err != nil {
		panic(err)
	}
}

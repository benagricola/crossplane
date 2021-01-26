/*
Copyright 2020 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/utils/pointer"

	"github.com/crossplane/crossplane-runtime/pkg/resource/fake"
	"github.com/crossplane/crossplane-runtime/pkg/test"
)

func TestPatchTypeReplacement(t *testing.T) {
	type args struct {
		comp CompositionSpec
	}

	type want struct {
		resources []ComposedTemplate
		err       error
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"NoCompositionPatchSets": {
			reason: "Patches defined on a composite resource should be applied correctly if no PatchSets are defined on the composition",
			args: args{
				comp: CompositionSpec{
					Resources: []ComposedTemplate{
						{
							Patches: []Patch{
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("metadata.name"),
								},
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("metadata.namespace"),
								},
							},
						},
					},
				},
			},
			want: want{
				resources: []ComposedTemplate{
					{
						Patches: []Patch{
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("metadata.name"),
							},
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("metadata.namespace"),
							},
						},
					},
				},
			},
		},
		"UndefinedPatchSet": {
			reason: "Should return error and not modify the patches field when referring to an undefined PatchSet",
			args: args{
				comp: CompositionSpec{
					Resources: []ComposedTemplate{{
						Patches: []Patch{
							{
								Type:         PatchTypePatchSet,
								PatchSetName: pointer.StringPtr("patch-set-1"),
							},
						},
					}},
				},
			},
			want: want{
				resources: []ComposedTemplate{{
					Patches: []Patch{
						{
							Type:         PatchTypePatchSet,
							PatchSetName: pointer.StringPtr("patch-set-1"),
						},
					},
				}},
				err: errors.Errorf(errFmtUndefinedPatchSet, "patch-set-1"),
			},
		},
		"DefinedPatchSets": {
			reason: "Should de-reference PatchSets defined on the Composition when referenced in a composed resource",
			args: args{
				comp: CompositionSpec{
					// PatchSets, existing patches and references
					// should output in the correct order.
					PatchSets: []PatchSet{
						{
							Name: "patch-set-1",
							Patches: []Patch{
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("metadata.namespace"),
								},
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("spec.parameters.test"),
								},
							},
						},
						{
							Name: "patch-set-2",
							Patches: []Patch{
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("metadata.annotations.patch-test-1"),
								},
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("metadata.annotations.patch-test-2"),
									Transforms: []Transform{{
										Type: TransformTypeMap,
										Map: &MapTransform{
											Pairs: map[string]string{
												"k-1": "v-1",
												"k-2": "v-2",
											},
										},
									}},
								},
							},
						},
					},
					Resources: []ComposedTemplate{
						{
							Patches: []Patch{
								{
									Type:         PatchTypePatchSet,
									PatchSetName: pointer.StringPtr("patch-set-2"),
								},
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("metadata.name"),
								},
								{
									Type:         PatchTypePatchSet,
									PatchSetName: pointer.StringPtr("patch-set-1"),
								},
							},
						},
						{
							Patches: []Patch{
								{
									Type:         PatchTypePatchSet,
									PatchSetName: pointer.StringPtr("patch-set-1"),
								},
							},
						},
					},
				},
			},
			want: want{
				err: nil,
				resources: []ComposedTemplate{
					{
						Patches: []Patch{
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("metadata.annotations.patch-test-1"),
							},
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("metadata.annotations.patch-test-2"),
								Transforms: []Transform{{
									Type: TransformTypeMap,
									Map: &MapTransform{
										Pairs: map[string]string{
											"k-1": "v-1",
											"k-2": "v-2",
										},
									},
								}},
							},
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("metadata.name"),
							},
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("metadata.namespace"),
							},
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("spec.parameters.test"),
							},
						},
					},
					{
						Patches: []Patch{
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("metadata.namespace"),
							},
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("spec.parameters.test"),
							},
						},
					},
				},
			},
		},
		"CombinePatchSet": {
			reason: "Should de-reference a referenced PatchSet with combine set",
			args: args{
				comp: CompositionSpec{
					PatchSets: []PatchSet{
						{
							Name: "combine-fields",
							Patches: []Patch{
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("spec.fieldOne"),
								},
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("spec.fieldTwo"),
								},
							},
						},
						{
							Name: "single-field",
							Patches: []Patch{
								{
									Type:          PatchTypeFromCompositeFieldPath,
									FromFieldPath: pointer.StringPtr("spec.fieldThree"),
								},
							},
						},
					},
					Resources: []ComposedTemplate{
						{
							Patches: []Patch{
								{
									Type:         PatchTypePatchSet,
									PatchSetName: pointer.StringPtr("combine-fields"),
									ToFieldPath:  pointer.StringPtr("spec.forProvider.out"),
									Combine: Combine{
										Type: CombineTypeString,
										String: &StringCombine{
											Format: "%s: %s",
										},
									},
								},
								{
									Type:         PatchTypePatchSet,
									PatchSetName: pointer.StringPtr("single-field"),
								},
							},
						},
					},
				},
			},
			want: want{
				err: nil,
				resources: []ComposedTemplate{
					{
						Patches: []Patch{
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("spec.fieldOne"),
								ToFieldPath:   pointer.StringPtr(combinerIdentifierString),
							},
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("spec.fieldTwo"),
								ToFieldPath:   pointer.StringPtr(combinerIdentifierString),
							},
							{
								Type:         PatchTypePatchSet,
								PatchSetName: pointer.StringPtr("combine-fields"),
								ToFieldPath:  pointer.StringPtr("spec.forProvider.out"),
								Combine: Combine{
									Type: CombineTypeString,
									String: &StringCombine{
										Format: "%s: %s",
									},
								},
							},
							{
								Type:          PatchTypeFromCompositeFieldPath,
								FromFieldPath: pointer.StringPtr("spec.fieldThree"),
							},
						},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := tc.args.comp.InlinePatchSets()

			if diff := cmp.Diff(tc.want.resources, tc.args.comp.Resources); diff != "" {
				t.Errorf("\n%s\nInlinePatchSets(b): -want, +got:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nInlinePatchSets(b): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestMapResolve(t *testing.T) {
	type args struct {
		m map[string]string
		i interface{}
	}
	type want struct {
		o   interface{}
		err error
	}

	cases := map[string]struct {
		args
		want
	}{
		"NonStringInput": {
			args: args{
				i: 5,
			},
			want: want{
				err: errors.Errorf(errFmtMapTypeNotSupported, "int"),
			},
		},
		"KeyNotFound": {
			args: args{
				i: "ola",
			},
			want: want{
				err: errors.Errorf(errFmtMapNotFound, "ola"),
			},
		},
		"Success": {
			args: args{
				m: map[string]string{"ola": "voila"},
				i: "ola",
			},
			want: want{
				o: "voila",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := (&MapTransform{Pairs: tc.m}).Resolve(tc.i)

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Resolve(b): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Resolve(b): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestMathResolve(t *testing.T) {
	m := int64(2)

	type args struct {
		multiplier *int64
		i          interface{}
	}
	type want struct {
		o   interface{}
		err error
	}

	cases := map[string]struct {
		args
		want
	}{
		"NoMultiplier": {
			args: args{
				i: 25,
			},
			want: want{
				err: errors.New(errMathNoMultiplier),
			},
		},
		"NonNumberInput": {
			args: args{
				multiplier: &m,
				i:          "ola",
			},
			want: want{
				err: errors.New(errMathInputNonNumber),
			},
		},
		"Success": {
			args: args{
				multiplier: &m,
				i:          3,
			},
			want: want{
				o: 3 * m,
			},
		},
		"SuccessInt64": {
			args: args{
				multiplier: &m,
				i:          int64(3),
			},
			want: want{
				o: 3 * m,
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := (&MathTransform{Multiply: tc.multiplier}).Resolve(tc.i)

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Resolve(b): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Resolve(b): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestStringResolve(t *testing.T) {

	type args struct {
		fmts string
		i    interface{}
	}
	type want struct {
		o   interface{}
		err error
	}

	cases := map[string]struct {
		args
		want
	}{
		"FmtString": {
			args: args{
				fmts: "verycool%s",
				i:    "thing",
			},
			want: want{
				o: "verycoolthing",
			},
		},
		"FmtInteger": {
			args: args{
				fmts: "the largest %d",
				i:    8,
			},
			want: want{
				o: "the largest 8",
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := (&StringTransform{Format: tc.fmts}).Resolve(tc.i)

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Resolve(b): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Resolve(b): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestConvertResolve(t *testing.T) {
	type args struct {
		ot string
		i  interface{}
	}
	type want struct {
		o   interface{}
		err error
	}

	cases := map[string]struct {
		args
		want
	}{
		"StringToBool": {
			args: args{
				i:  "true",
				ot: ConvertTransformTypeBool,
			},
			want: want{
				o: true,
			},
		},
		"InputTypeNotSupported": {
			args: args{
				i:  []int{64},
				ot: ConvertTransformTypeString,
			},
			want: want{
				err: errors.Errorf(errFmtConvertInputTypeNotSupported, reflect.TypeOf([]int{}).Kind().String()),
			},
		},
		"ConversionPairNotSupported": {
			args: args{
				i:  "[64]",
				ot: "[]int",
			},
			want: want{
				err: errors.Errorf(errFmtConversionPairNotSupported, "string", "[]int"),
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := (&ConvertTransform{ToType: tc.args.ot}).Resolve(tc.i)

			if diff := cmp.Diff(tc.want.o, got); diff != "" {
				t.Errorf("Resolve(b): -want, +got:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("Resolve(b): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestPatchApply(t *testing.T) {
	now := metav1.NewTime(time.Unix(0, 0))
	lpt := fake.ConnectionDetailsLastPublishedTimer{
		Time: &now,
	}

	type args struct {
		patch Patch
		cp    *fake.Composite
		cd    *fake.Composed
		tmp   *[]interface{}
	}
	type want struct {
		cp  *fake.Composite
		cd  *fake.Composed
		err error
		tmp *[]interface{}
	}

	cases := map[string]struct {
		reason string
		args
		want
	}{
		"InvalidCompositeFieldPathPatch": {
			reason: "Should return error when required fields not passed to applyFromCompositeFieldPatch",
			args: args{
				patch: Patch{
					Type: PatchTypeFromCompositeFieldPath,
				},
				cp: &fake.Composite{
					ConnectionDetailsLastPublishedTimer: lpt,
				},
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{Name: "cd"}},
			},
			want: want{
				err: errors.Errorf(errFmtRequiredField, "FromFieldPath", PatchTypeFromCompositeFieldPath),
			},
		},
		"InvalidPatchType": {
			reason: "Should return an error if an invalid patch type is specified",
			args: args{
				patch: Patch{
					Type: "invalid-patchtype",
				},
				cp: &fake.Composite{
					ConnectionDetailsLastPublishedTimer: lpt,
				},
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{Name: "cd"}},
			},
			want: want{
				err: errors.Errorf(errFmtInvalidPatchType, "invalid-patchtype"),
			},
		},
		"ValidCompositeFieldPathPatch": {
			reason: "Should correctly apply a CompositeFieldPathPatch with valid settings",
			args: args{
				patch: Patch{
					Type:          PatchTypeFromCompositeFieldPath,
					FromFieldPath: pointer.StringPtr("objectMeta.labels"),
					ToFieldPath:   pointer.StringPtr("objectMeta.labels"),
				},
				cp: &fake.Composite{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cp",
						Labels: map[string]string{
							"Test": "blah",
						},
					},
					ConnectionDetailsLastPublishedTimer: lpt,
				},
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{Name: "cd"},
				},
			},
			want: want{
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{Name: "cd", Labels: map[string]string{
						"Test": "blah",
					}},
				},
				tmp: &([]interface{}{}),
				err: nil,
			},
		},
		"DefaultToFieldCompositeFieldPathPatch": {
			reason: "Should correctly default the ToFieldPath value if not specified.",
			args: args{
				patch: Patch{
					Type:          PatchTypeFromCompositeFieldPath,
					FromFieldPath: pointer.StringPtr("objectMeta.labels"),
				},
				cp: &fake.Composite{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cp",
						Labels: map[string]string{
							"Test": "blah",
						},
					},
					ConnectionDetailsLastPublishedTimer: lpt,
				},
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{Name: "cd"},
				},
			},
			want: want{
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{Name: "cd", Labels: map[string]string{
						"Test": "blah",
					}},
				},
				err: nil,
			},
		},
		"ValidCompositeToTemporaryPatch": {
			reason: "Should return the patch value rather than applying to target if toFieldPath is an empty string",
			args: args{
				patch: Patch{
					Type:          PatchTypeFromCompositeFieldPath,
					FromFieldPath: pointer.StringPtr("objectMeta.labels['Test']"),
					ToFieldPath:   pointer.StringPtr(""),
				},
				cp: &fake.Composite{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cp",
						Labels: map[string]string{
							"Test": "blah",
						},
					},
					ConnectionDetailsLastPublishedTimer: lpt,
				},
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{Name: "cd"},
				},
			},
			want: want{
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{Name: "cd"},
				},
				tmp: &([]interface{}{"blah"}),
				err: nil,
			},
		},
		"PatchSetNoCombine": {
			reason: "Should return no error if patchSet is no-op",
			args: args{
				patch: Patch{
					Type:         PatchTypePatchSet,
					PatchSetName: pointer.StringPtr("blah"),
				},
				cp: &fake.Composite{
					ConnectionDetailsLastPublishedTimer: lpt,
				},
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{Name: "cd"}},
			},
			want: want{
				err: nil,
			},
		},
		"PatchSetCombineString": {
			reason: "Should combine temporary values using string combine",
			args: args{
				patch: Patch{
					Type:         PatchTypePatchSet,
					PatchSetName: pointer.StringPtr("blah"),
					ToFieldPath:  pointer.StringPtr("objectMeta.labels['Test']"),
					Combine: Combine{
						Type: CombineTypeString,
						String: &StringCombine{
							Format: "%s-%s",
						},
					},
				},
				tmp: &[]interface{}{
					"test1",
					"test2",
				},
				cp: &fake.Composite{
					ConnectionDetailsLastPublishedTimer: lpt,
				},
				cd: &fake.Composed{ObjectMeta: metav1.ObjectMeta{Name: "cd"}},
			},
			want: want{
				err: nil,
				cd: &fake.Composed{
					ObjectMeta: metav1.ObjectMeta{Name: "cd", Labels: map[string]string{
						"Test": "test1-test2",
					}},
				},
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if tc.args.tmp == nil {
				tc.args.tmp = &([]interface{}{})
			}
			ncp := tc.args.cp.DeepCopyObject()
			err := tc.args.patch.Apply(ncp, tc.args.cd, tc.args.tmp)
			if tc.want.cp != nil {
				if diff := cmp.Diff(tc.want.cp, ncp); diff != "" {
					t.Errorf("\n%s\nApply(cp): -want, +got:\n%s", tc.reason, diff)
				}
			}
			if tc.want.cd != nil {
				if diff := cmp.Diff(tc.want.cd, tc.args.cd); diff != "" {
					t.Errorf("\n%s\nApply(cd): -want, +got:\n%s", tc.reason, diff)
				}
			}
			if tc.want.tmp != nil {
				if diff := cmp.Diff(tc.want.tmp, tc.args.tmp); diff != "" {
					t.Errorf("\n%s\nApply(tmp): -want, +got:\n%s", tc.reason, diff)
				}
			}
			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nApply(err): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

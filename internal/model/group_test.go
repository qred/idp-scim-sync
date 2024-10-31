package model

import (
	"bytes"
	"encoding/gob"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestGroup_SetHashCode(t *testing.T) {
	tests := []struct {
		name  string
		group *Group
		want  *Group
	}{
		{
			name: "success",
			group: &Group{
				IPID:     "1",
				SCIMID:   "1",
				Name:     "group 1",
				Email:    "user.1@mail.com",
				HashCode: "test",
			},
			want: &Group{
				IPID:  "1",
				Name:  "group 1",
				Email: "user.1@mail.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.group.SetHashCode()
			tt.want.SetHashCode()

			got := tt.group.HashCode
			if got != tt.want.HashCode {
				t.Errorf("Group.SetHashCode() = %s, want %s", got, tt.want.HashCode)
			}
		})
	}
}

func TestGroup_GobEncode(t *testing.T) {
	tests := []struct {
		name   string
		toTest *Group
	}{
		{
			name:   "empty",
			toTest: &Group{},
		},
		{
			name: "filled",
			toTest: &Group{
				IPID:     "1",
				SCIMID:   "1",
				Name:     "group",
				Email:    "user.1@mail.com",
				HashCode: "this should not be encoded",
			},
		},
		{
			name: "filled partial",
			toTest: &Group{
				IPID:     "1",
				Name:     "group",
				HashCode: "this should not be encoded",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)

			if err := enc.Encode(tt.toTest); err != nil {
				t.Errorf("Group.MarshalBinary() error = %v", err)
			}

			dec := gob.NewDecoder(&buf)
			var got Group
			if err := dec.Decode(&got); err != nil {
				t.Errorf("Group.UnmarshalBinary() error = %v", err)
			}

			// SCIMID is not exported, so it will not be encoded
			// HashCode is not exported, so it will not be encoded
			expected := Group{
				IPID:  tt.toTest.IPID,
				Name:  tt.toTest.Name,
				Email: tt.toTest.Email,
			}

			sort := func(x, y string) bool { return x > y }
			if diff := cmp.Diff(expected, got, cmpopts.SortSlices(sort)); diff != "" {
				t.Errorf("mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestGroupsResult_GobEncode(t *testing.T) {
	tests := []struct {
		name   string
		toTest *GroupsResult
	}{
		{
			name:   "empty",
			toTest: &GroupsResult{},
		},
		{
			name: "filled",
			toTest: &GroupsResult{
				Items:    1,
				HashCode: "hashcode",
				Resources: []*Group{
					{
						IPID:     "1",
						SCIMID:   "1",
						Name:     "group",
						Email:    "user.1@mail.com",
						HashCode: "hashcode",
					},
				},
			},
		},
		{
			name: "filled partial",
			toTest: &GroupsResult{
				Items:    1,
				HashCode: "test",
				Resources: []*Group{
					{
						IPID:     "1",
						Name:     "group",
						HashCode: "test",
					},
				},
			},
		},
		{
			name: "filled partial 2",
			toTest: &GroupsResult{
				Items:    2,
				HashCode: "test",
				Resources: []*Group{
					{
						IPID:     "1",
						Name:     "group",
						HashCode: "test",
					},
					{
						IPID:     "2",
						Name:     "group",
						HashCode: "test",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			enc := gob.NewEncoder(buf)

			if err := enc.Encode(tt.toTest); err != nil {
				t.Errorf("User.GobEncode() error = %v", err)
			}

			dec := gob.NewDecoder(buf)
			var got GroupsResult
			if err := dec.Decode(&got); err != nil {
				t.Errorf("User.GobEncode() error = %v", err)
			}

			// SCIMID is not exported, so it will not be encoded
			// HashCode is not exported, so it will not be encoded
			var expectedGroups []*Group
			for _, g := range tt.toTest.Resources {
				expectedGroups = append(expectedGroups, &Group{
					IPID:  g.IPID,
					Name:  g.Name,
					Email: g.Email,
				})
			}

			expected := GroupsResult{
				Items:     tt.toTest.Items,
				Resources: expectedGroups,
			}

			sort := func(x, y string) bool { return x > y }
			if diff := cmp.Diff(expected, got, cmpopts.SortSlices(sort)); diff != "" {
				t.Errorf("mismatch (-expected +got):\n%s", diff)
			}
		})
	}
}

func TestGroupsResult_SetHashCode(t *testing.T) {
	g1 := &Group{IPID: "1", SCIMID: "1", Name: "group", Email: "group.1@mail.com"}
	g2 := &Group{IPID: "2", SCIMID: "2", Name: "group", Email: "group.2@mail.com"}
	g3 := &Group{IPID: "3", SCIMID: "3", Name: "group", Email: "group.3@mail.com"}

	g1.SetHashCode()
	g2.SetHashCode()
	g3.SetHashCode()

	gr1 := GroupsResult{
		Items:     3,
		Resources: []*Group{g1, g2, g3},
	}
	gr1.SetHashCode()

	gr2 := GroupsResult{
		Items:     3,
		Resources: []*Group{g2, g3, g1},
	}
	gr2.SetHashCode()

	gr3 := GroupsResult{
		Items:     3,
		Resources: []*Group{g3, g2, g1},
	}
	gr3.SetHashCode()

	gr4 := MergeGroupsResult(&gr2, &gr1, &gr3)
	gr4.SetHashCode()
	gr5 := MergeGroupsResult(&gr3, &gr2, &gr1)
	gr5.SetHashCode()

	if gr1.HashCode != gr2.HashCode {
		t.Errorf("GroupsResult.HashCode should be equal")
	}
	if gr1.HashCode != gr3.HashCode {
		t.Errorf("GroupsResult.HashCode should be equal")
	}
	if gr2.HashCode != gr3.HashCode {
		t.Errorf("GroupsResult.HashCode should be equal")
	}

	if gr5.HashCode != gr4.HashCode {
		t.Errorf("GroupsResult.HashCode should be equal: gr5-> %s, gr4-> %s", gr5.HashCode, gr4.HashCode)
	}
}

func TestGroupsResult_MarshalJSON(t *testing.T) {
	type fields struct {
		Items     int
		HashCode  string
		Resources []*Group
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name:   "empty",
			fields: fields{},
			want: []byte(`{
  "items": 0,
  "resources": []
}`),
			wantErr: false,
		},
		{
			name: "success",
			fields: fields{
				Items:    1,
				HashCode: "test",
				Resources: []*Group{
					{
						IPID:     "1",
						SCIMID:   "1",
						Name:     "group",
						HashCode: "1111",
					},
				},
			},
			want: []byte(`{
  "items": 1,
  "hashCode": "test",
  "resources": [
    {
      "ipid": "1",
      "scimid": "1",
      "name": "group",
      "hashCode": "1111"
    }
  ]
}`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gr := &GroupsResult{
				Items:     tt.fields.Items,
				HashCode:  tt.fields.HashCode,
				Resources: tt.fields.Resources,
			}
			got, err := gr.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("GroupsResult.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GroupsResult.MarshalJSON() = %s, want %s", string(got), string(tt.want))
			}
		})
	}
}

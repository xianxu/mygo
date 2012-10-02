package gassandra

import (
	"fmt"
)

type IndexOperator int32

var (
	IndexOperatorEq  = IndexOperator(1)
	IndexOperatorGt  = IndexOperator(3)
	IndexOperatorGte = IndexOperator(2)
	IndexOperatorLt  = IndexOperator(5)
	IndexOperatorLte = IndexOperator(4)
)

func (e IndexOperator) String() string {
	switch e {
	case IndexOperatorEq:
		return "IndexOperatorEq"
	case IndexOperatorGt:
		return "IndexOperatorGt"
	case IndexOperatorGte:
		return "IndexOperatorGte"
	case IndexOperatorLt:
		return "IndexOperatorLt"
	case IndexOperatorLte:
		return "IndexOperatorLte"
	}
	return fmt.Sprintf("Unknown value for IndexOperatorLte: %d", e)
}

type CqlResultType int32

var (
	CqlResultTypeInt  = CqlResultType(3)
	CqlResultTypeRows = CqlResultType(1)
	CqlResultTypeVoid = CqlResultType(2)
)

func (e CqlResultType) String() string {
	switch e {
	case CqlResultTypeInt:
		return "CqlResultTypeInt"
	case CqlResultTypeRows:
		return "CqlResultTypeRows"
	case CqlResultTypeVoid:
		return "CqlResultTypeVoid"
	}
	return fmt.Sprintf("Unknown value for CqlResultTypeVoid: %d", e)
}

type IndexType int32

var (
	IndexTypeKeys = IndexType(1)
)

func (e IndexType) String() string {
	switch e {
	case IndexTypeKeys:
		return "IndexTypeKeys"
	}
	return fmt.Sprintf("Unknown value for IndexTypeKeys: %d", e)
}

type ConsistencyLevel int32

var (
	ConsistencyLevelAll         = ConsistencyLevel(5)
	ConsistencyLevelAny         = ConsistencyLevel(6)
	ConsistencyLevelEachQuorum  = ConsistencyLevel(4)
	ConsistencyLevelLocalQuorum = ConsistencyLevel(3)
	ConsistencyLevelOne         = ConsistencyLevel(1)
	ConsistencyLevelQuorum      = ConsistencyLevel(2)
	ConsistencyLevelThree       = ConsistencyLevel(8)
	ConsistencyLevelTwo         = ConsistencyLevel(7)
)

func (e ConsistencyLevel) String() string {
	switch e {
	case ConsistencyLevelAll:
		return "ConsistencyLevelAll"
	case ConsistencyLevelAny:
		return "ConsistencyLevelAny"
	case ConsistencyLevelEachQuorum:
		return "ConsistencyLevelEachQuorum"
	case ConsistencyLevelLocalQuorum:
		return "ConsistencyLevelLocalQuorum"
	case ConsistencyLevelOne:
		return "ConsistencyLevelOne"
	case ConsistencyLevelQuorum:
		return "ConsistencyLevelQuorum"
	case ConsistencyLevelThree:
		return "ConsistencyLevelThree"
	case ConsistencyLevelTwo:
		return "ConsistencyLevelTwo"
	}
	return fmt.Sprintf("Unknown value for ConsistencyLevelTwo: %d", e)
}

type Compression int32

var (
	CompressionGzip = Compression(1)
	CompressionNone = Compression(2)
)

func (e Compression) String() string {
	switch e {
	case CompressionGzip:
		return "CompressionGzip"
	case CompressionNone:
		return "CompressionNone"
	}
	return fmt.Sprintf("Unknown value for CompressionNone: %d", e)
}

type Column struct {
	Name      []byte `thrift:"1,required" json:"name"`
	Value     []byte `thrift:"2" json:"value"`
	Timestamp int64  `thrift:"3" json:"timestamp"`
	Ttl       int32  `thrift:"4" json:"ttl"`
}

type SuperColumn struct {
	Name    []byte    `thrift:"1,required" json:"name"`
	Columns []*Column `thrift:"2,required" json:"columns"`
}

type CounterColumn struct {
	Name  []byte `thrift:"1,required" json:"name"`
	Value int64  `thrift:"2,required" json:"value"`
}

type CounterSuperColumn struct {
	Name    []byte           `thrift:"1,required" json:"name"`
	Columns []*CounterColumn `thrift:"2,required" json:"columns"`
}

type ColumnOrSuperColumn struct {
	Column             *Column             `thrift:"1" json:"column"`
	SuperColumn        *SuperColumn        `thrift:"2" json:"super_column"`
	CounterColumn      *CounterColumn      `thrift:"3" json:"counter_column"`
	CounterSuperColumn *CounterSuperColumn `thrift:"4" json:"counter_super_column"`
}

type NotFoundException struct {

}

func (e *NotFoundException) Error() string {
	return fmt.Sprintf("NotFoundException{}")
}

type InvalidRequestException struct {
	Why string `thrift:"1,required" json:"why"`
}

func (e *InvalidRequestException) Error() string {
	return fmt.Sprintf("InvalidRequestException{Why:%+v}", e.Why)
}

type UnavailableException struct {

}

func (e *UnavailableException) Error() string {
	return fmt.Sprintf("UnavailableException{}")
}

type TimedOutException struct {

}

func (e *TimedOutException) Error() string {
	return fmt.Sprintf("TimedOutException{}")
}

type AuthenticationException struct {
	Why string `thrift:"1,required" json:"why"`
}

func (e *AuthenticationException) Error() string {
	return fmt.Sprintf("AuthenticationException{Why:%+v}", e.Why)
}

type AuthorizationException struct {
	Why string `thrift:"1,required" json:"why"`
}

func (e *AuthorizationException) Error() string {
	return fmt.Sprintf("AuthorizationException{Why:%+v}", e.Why)
}

type SchemaDisagreementException struct {

}

func (e *SchemaDisagreementException) Error() string {
	return fmt.Sprintf("SchemaDisagreementException{}")
}

type ColumnParent struct {
	ColumnFamily string `thrift:"3,required" json:"column_family"`
	SuperColumn  []byte `thrift:"4" json:"super_column"`
}

type ColumnPath struct {
	ColumnFamily string `thrift:"3,required" json:"column_family"`
	SuperColumn  []byte `thrift:"4" json:"super_column"`
	Column       []byte `thrift:"5" json:"column"`
}

type SliceRange struct {
	Start    []byte `thrift:"1,required" json:"start"`
	Finish   []byte `thrift:"2,required" json:"finish"`
	Reversed bool   `thrift:"3,required" json:"reversed"`
	Count    int32  `thrift:"4,required" json:"count"`
}

type SlicePredicate struct {
	ColumnNames [][]byte    `thrift:"1" json:"column_names"`
	SliceRange  *SliceRange `thrift:"2" json:"slice_range"`
}

type IndexExpression struct {
	ColumnName []byte        `thrift:"1,required" json:"column_name"`
	Op         IndexOperator `thrift:"2,required" json:"op"`
	Value      []byte        `thrift:"3,required" json:"value"`
}

type IndexClause struct {
	Expressions []*IndexExpression `thrift:"1,required" json:"expressions"`
	StartKey    []byte             `thrift:"2,required" json:"start_key"`
	Count       int32              `thrift:"3,required" json:"count"`
}

type KeyRange struct {
	StartKey   []byte `thrift:"1" json:"start_key"`
	EndKey     []byte `thrift:"2" json:"end_key"`
	StartToken string `thrift:"3" json:"start_token"`
	EndToken   string `thrift:"4" json:"end_token"`
	Count      int32  `thrift:"5,required" json:"count"`
}

type KeySlice struct {
	Key     []byte                 `thrift:"1,required" json:"key"`
	Columns []*ColumnOrSuperColumn `thrift:"2,required" json:"columns"`
}

type KeyCount struct {
	Key   []byte `thrift:"1,required" json:"key"`
	Count int32  `thrift:"2,required" json:"count"`
}

type Deletion struct {
	Timestamp   int64           `thrift:"1" json:"timestamp"`
	SuperColumn []byte          `thrift:"2" json:"super_column"`
	Predicate   *SlicePredicate `thrift:"3" json:"predicate"`
}

type Mutation struct {
	ColumnOrSupercolumn *ColumnOrSuperColumn `thrift:"1" json:"column_or_supercolumn"`
	Deletion            *Deletion            `thrift:"2" json:"deletion"`
}

type TokenRange struct {
	StartToken string   `thrift:"1,required" json:"start_token"`
	EndToken   string   `thrift:"2,required" json:"end_token"`
	Endpoints  []string `thrift:"3,required" json:"endpoints"`
}

type AuthenticationRequest struct {
	Credentials map[string]string `thrift:"1,required" json:"credentials"`
}

type ColumnDef struct {
	Name            []byte    `thrift:"1,required" json:"name"`
	ValidationClass string    `thrift:"2,required" json:"validation_class"`
	IndexType       IndexType `thrift:"3" json:"index_type"`
	IndexName       string    `thrift:"4" json:"index_name"`
}

type CfDef struct {
	Keyspace                     string       `thrift:"1,required" json:"keyspace"`
	Name                         string       `thrift:"2,required" json:"name"`
	ColumnType                   string       `thrift:"3" json:"column_type"`
	ComparatorType               string       `thrift:"5" json:"comparator_type"`
	SubcomparatorType            string       `thrift:"6" json:"subcomparator_type"`
	Comment                      string       `thrift:"8" json:"comment"`
	RowCacheSize                 float64      `thrift:"9" json:"row_cache_size"`
	KeyCacheSize                 float64      `thrift:"11" json:"key_cache_size"`
	ReadRepairChance             float64      `thrift:"12" json:"read_repair_chance"`
	ColumnMetadata               []*ColumnDef `thrift:"13" json:"column_metadata"`
	GcGraceSeconds               int32        `thrift:"14" json:"gc_grace_seconds"`
	DefaultValidationClass       string       `thrift:"15" json:"default_validation_class"`
	Id                           int32        `thrift:"16" json:"id"`
	MinCompactionThreshold       int32        `thrift:"17" json:"min_compaction_threshold"`
	MaxCompactionThreshold       int32        `thrift:"18" json:"max_compaction_threshold"`
	RowCacheSavePeriodInSeconds  int32        `thrift:"19" json:"row_cache_save_period_in_seconds"`
	KeyCacheSavePeriodInSeconds  int32        `thrift:"20" json:"key_cache_save_period_in_seconds"`
	MemtableFlushAfterMins       int32        `thrift:"21" json:"memtable_flush_after_mins"`
	MemtableThroughputInMb       int32        `thrift:"22" json:"memtable_throughput_in_mb"`
	MemtableOperationsInMillions float64      `thrift:"23" json:"memtable_operations_in_millions"`
	ReplicateOnWrite             bool         `thrift:"24" json:"replicate_on_write"`
	MergeShardsChance            float64      `thrift:"25" json:"merge_shards_chance"`
	KeyValidationClass           string       `thrift:"26" json:"key_validation_class"`
	RowCacheProvider             string       `thrift:"27" json:"row_cache_provider"`
	KeyAlias                     []byte       `thrift:"28" json:"key_alias"`
}

type KsDef struct {
	Name              string            `thrift:"1,required" json:"name"`
	StrategyClass     string            `thrift:"2,required" json:"strategy_class"`
	StrategyOptions   map[string]string `thrift:"3" json:"strategy_options"`
	ReplicationFactor int32             `thrift:"4" json:"replication_factor"`
	CfDefs            []*CfDef          `thrift:"5,required" json:"cf_defs"`
	DurableWrites     bool              `thrift:"6" json:"durable_writes"`
}

type CqlRow struct {
	Key     []byte    `thrift:"1,required" json:"key"`
	Columns []*Column `thrift:"2,required" json:"columns"`
}

type CqlResult struct {
	Type CqlResultType `thrift:"1,required" json:"type"`
	Rows []*CqlRow     `thrift:"2" json:"rows"`
	Num  int32         `thrift:"3" json:"num"`
}

type Cassandra interface {
	GetRangeSlices(ColumnParent *ColumnParent, Predicate *SlicePredicate, Range *KeyRange, ConsistencyLevel ConsistencyLevel) ([]*KeySlice, error)
	SetKeyspace(Keyspace string) error
	SystemDropKeyspace(Keyspace string) (string, error)
	SystemDropColumnFamily(ColumnFamily string) (string, error)
	SystemAddColumnFamily(CfDef *CfDef) (string, error)
	DescribeSnitch() (string, error)
	MultigetCount(Keys [][]byte, ColumnParent *ColumnParent, Predicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) (map[string]int32, error)
	DescribePartitioner() (string, error)
	DescribeSplits(CfName string, StartToken string, EndToken string, KeysPerSplit int32) ([]string, error)
	RemoveCounter(Key []byte, Path *ColumnPath, ConsistencyLevel ConsistencyLevel) error
	DescribeSchemaVersions() (map[string][]string, error)
	Add(Key []byte, ColumnParent *ColumnParent, Column *CounterColumn, ConsistencyLevel ConsistencyLevel) error
	GetSlice(Key []byte, ColumnParent *ColumnParent, Predicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) ([]*ColumnOrSuperColumn, error)
	GetCount(Key []byte, ColumnParent *ColumnParent, Predicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) (int32, error)
	DescribeVersion() (string, error)
	MultigetSlice(Keys [][]byte, ColumnParent *ColumnParent, Predicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) (map[string][]*ColumnOrSuperColumn, error)
	SystemUpdateColumnFamily(CfDef *CfDef) (string, error)
	Truncate(Cfname string) error
	Get(Key []byte, ColumnPath *ColumnPath, ConsistencyLevel ConsistencyLevel) (*ColumnOrSuperColumn, error)
	BatchMutate(MutationMap map[string]map[string][]*Mutation, ConsistencyLevel ConsistencyLevel) error
	SystemUpdateKeyspace(KsDef *KsDef) (string, error)
	SystemAddKeyspace(KsDef *KsDef) (string, error)
	GetIndexedSlices(ColumnParent *ColumnParent, IndexClause *IndexClause, ColumnPredicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) ([]*KeySlice, error)
	Insert(Key []byte, ColumnParent *ColumnParent, Column *Column, ConsistencyLevel ConsistencyLevel) error
	DescribeKeyspace(Keyspace string) (*KsDef, error)
	ExecuteCqlQuery(Query []byte, Compression Compression) (*CqlResult, error)
	DescribeRing(Keyspace string) ([]*TokenRange, error)
	Remove(Key []byte, ColumnPath *ColumnPath, Timestamp int64, ConsistencyLevel ConsistencyLevel) error
	DescribeKeyspaces() ([]*KsDef, error)
	Login(AuthRequest *AuthenticationRequest) error
	DescribeClusterName() (string, error)
}

type CassandraGetRangeSlicesRequest struct {
	ColumnParent     *ColumnParent    `thrift:"1,required" json:"column_parent"`
	Predicate        *SlicePredicate  `thrift:"2,required" json:"predicate"`
	Range            *KeyRange        `thrift:"3,required" json:"range"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraGetRangeSlicesResponse struct {
	Value []*KeySlice              `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
	Ue    *UnavailableException    `thrift:"2" json:"ue"`
	Te    *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraSetKeyspaceRequest struct {
	Keyspace string `thrift:"1,required" json:"keyspace"`
}

type CassandraSetKeyspaceResponse struct {
	Ire *InvalidRequestException `thrift:"1" json:"ire"`
}

type CassandraSystemDropKeyspaceRequest struct {
	Keyspace string `thrift:"1,required" json:"keyspace"`
}

type CassandraSystemDropKeyspaceResponse struct {
	Value string                       `thrift:"0" json:"value"`
	Ire   *InvalidRequestException     `thrift:"1" json:"ire"`
	Sde   *SchemaDisagreementException `thrift:"2" json:"sde"`
}

type CassandraSystemDropColumnFamilyRequest struct {
	ColumnFamily string `thrift:"1,required" json:"column_family"`
}

type CassandraSystemDropColumnFamilyResponse struct {
	Value string                       `thrift:"0" json:"value"`
	Ire   *InvalidRequestException     `thrift:"1" json:"ire"`
	Sde   *SchemaDisagreementException `thrift:"2" json:"sde"`
}

type CassandraSystemAddColumnFamilyRequest struct {
	CfDef *CfDef `thrift:"1,required" json:"cf_def"`
}

type CassandraSystemAddColumnFamilyResponse struct {
	Value string                       `thrift:"0" json:"value"`
	Ire   *InvalidRequestException     `thrift:"1" json:"ire"`
	Sde   *SchemaDisagreementException `thrift:"2" json:"sde"`
}

type CassandraDescribeSnitchRequest struct {

}

type CassandraDescribeSnitchResponse struct {
	Value string `thrift:"0" json:"value"`
}

type CassandraMultigetCountRequest struct {
	Keys             [][]byte         `thrift:"1,required" json:"keys"`
	ColumnParent     *ColumnParent    `thrift:"2,required" json:"column_parent"`
	Predicate        *SlicePredicate  `thrift:"3,required" json:"predicate"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraMultigetCountResponse struct {
	Value map[string]int32         `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
	Ue    *UnavailableException    `thrift:"2" json:"ue"`
	Te    *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraDescribePartitionerRequest struct {

}

type CassandraDescribePartitionerResponse struct {
	Value string `thrift:"0" json:"value"`
}

type CassandraDescribeSplitsRequest struct {
	CfName       string `thrift:"1,required" json:"cfName"`
	StartToken   string `thrift:"2,required" json:"start_token"`
	EndToken     string `thrift:"3,required" json:"end_token"`
	KeysPerSplit int32  `thrift:"4,required" json:"keys_per_split"`
}

type CassandraDescribeSplitsResponse struct {
	Value []string                 `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
}

type CassandraRemoveCounterRequest struct {
	Key              []byte           `thrift:"1,required" json:"key"`
	Path             *ColumnPath      `thrift:"2,required" json:"path"`
	ConsistencyLevel ConsistencyLevel `thrift:"3,required" json:"consistency_level"`
}

type CassandraRemoveCounterResponse struct {
	Ire *InvalidRequestException `thrift:"1" json:"ire"`
	Ue  *UnavailableException    `thrift:"2" json:"ue"`
	Te  *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraDescribeSchemaVersionsRequest struct {

}

type CassandraDescribeSchemaVersionsResponse struct {
	Value map[string][]string      `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
}

type CassandraAddRequest struct {
	Key              []byte           `thrift:"1,required" json:"key"`
	ColumnParent     *ColumnParent    `thrift:"2,required" json:"column_parent"`
	Column           *CounterColumn   `thrift:"3,required" json:"column"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraAddResponse struct {
	Ire *InvalidRequestException `thrift:"1" json:"ire"`
	Ue  *UnavailableException    `thrift:"2" json:"ue"`
	Te  *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraGetSliceRequest struct {
	Key              []byte           `thrift:"1,required" json:"key"`
	ColumnParent     *ColumnParent    `thrift:"2,required" json:"column_parent"`
	Predicate        *SlicePredicate  `thrift:"3,required" json:"predicate"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraGetSliceResponse struct {
	Value []*ColumnOrSuperColumn   `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
	Ue    *UnavailableException    `thrift:"2" json:"ue"`
	Te    *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraGetCountRequest struct {
	Key              []byte           `thrift:"1,required" json:"key"`
	ColumnParent     *ColumnParent    `thrift:"2,required" json:"column_parent"`
	Predicate        *SlicePredicate  `thrift:"3,required" json:"predicate"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraGetCountResponse struct {
	Value int32                    `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
	Ue    *UnavailableException    `thrift:"2" json:"ue"`
	Te    *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraDescribeVersionRequest struct {

}

type CassandraDescribeVersionResponse struct {
	Value string `thrift:"0" json:"value"`
}

type CassandraMultigetSliceRequest struct {
	Keys             [][]byte         `thrift:"1,required" json:"keys"`
	ColumnParent     *ColumnParent    `thrift:"2,required" json:"column_parent"`
	Predicate        *SlicePredicate  `thrift:"3,required" json:"predicate"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraMultigetSliceResponse struct {
	Value map[string][]*ColumnOrSuperColumn `thrift:"0" json:"value"`
	Ire   *InvalidRequestException          `thrift:"1" json:"ire"`
	Ue    *UnavailableException             `thrift:"2" json:"ue"`
	Te    *TimedOutException                `thrift:"3" json:"te"`
}

type CassandraSystemUpdateColumnFamilyRequest struct {
	CfDef *CfDef `thrift:"1,required" json:"cf_def"`
}

type CassandraSystemUpdateColumnFamilyResponse struct {
	Value string                       `thrift:"0" json:"value"`
	Ire   *InvalidRequestException     `thrift:"1" json:"ire"`
	Sde   *SchemaDisagreementException `thrift:"2" json:"sde"`
}

type CassandraTruncateRequest struct {
	Cfname string `thrift:"1,required" json:"cfname"`
}

type CassandraTruncateResponse struct {
	Ire *InvalidRequestException `thrift:"1" json:"ire"`
	Ue  *UnavailableException    `thrift:"2" json:"ue"`
}

type CassandraGetRequest struct {
	Key              []byte           `thrift:"1,required" json:"key"`
	ColumnPath       *ColumnPath      `thrift:"2,required" json:"column_path"`
	ConsistencyLevel ConsistencyLevel `thrift:"3,required" json:"consistency_level"`
}

type CassandraGetResponse struct {
	Value *ColumnOrSuperColumn     `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
	Nfe   *NotFoundException       `thrift:"2" json:"nfe"`
	Ue    *UnavailableException    `thrift:"3" json:"ue"`
	Te    *TimedOutException       `thrift:"4" json:"te"`
}

type CassandraBatchMutateRequest struct {
	MutationMap      map[string]map[string][]*Mutation `thrift:"1,required" json:"mutation_map"`
	ConsistencyLevel ConsistencyLevel                  `thrift:"2,required" json:"consistency_level"`
}

type CassandraBatchMutateResponse struct {
	Ire *InvalidRequestException `thrift:"1" json:"ire"`
	Ue  *UnavailableException    `thrift:"2" json:"ue"`
	Te  *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraSystemUpdateKeyspaceRequest struct {
	KsDef *KsDef `thrift:"1,required" json:"ks_def"`
}

type CassandraSystemUpdateKeyspaceResponse struct {
	Value string                       `thrift:"0" json:"value"`
	Ire   *InvalidRequestException     `thrift:"1" json:"ire"`
	Sde   *SchemaDisagreementException `thrift:"2" json:"sde"`
}

type CassandraSystemAddKeyspaceRequest struct {
	KsDef *KsDef `thrift:"1,required" json:"ks_def"`
}

type CassandraSystemAddKeyspaceResponse struct {
	Value string                       `thrift:"0" json:"value"`
	Ire   *InvalidRequestException     `thrift:"1" json:"ire"`
	Sde   *SchemaDisagreementException `thrift:"2" json:"sde"`
}

type CassandraGetIndexedSlicesRequest struct {
	ColumnParent     *ColumnParent    `thrift:"1,required" json:"column_parent"`
	IndexClause      *IndexClause     `thrift:"2,required" json:"index_clause"`
	ColumnPredicate  *SlicePredicate  `thrift:"3,required" json:"column_predicate"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraGetIndexedSlicesResponse struct {
	Value []*KeySlice              `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
	Ue    *UnavailableException    `thrift:"2" json:"ue"`
	Te    *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraInsertRequest struct {
	Key              []byte           `thrift:"1,required" json:"key"`
	ColumnParent     *ColumnParent    `thrift:"2,required" json:"column_parent"`
	Column           *Column          `thrift:"3,required" json:"column"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraInsertResponse struct {
	Ire *InvalidRequestException `thrift:"1" json:"ire"`
	Ue  *UnavailableException    `thrift:"2" json:"ue"`
	Te  *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraDescribeKeyspaceRequest struct {
	Keyspace string `thrift:"1,required" json:"keyspace"`
}

type CassandraDescribeKeyspaceResponse struct {
	Value *KsDef                   `thrift:"0" json:"value"`
	Nfe   *NotFoundException       `thrift:"1" json:"nfe"`
	Ire   *InvalidRequestException `thrift:"2" json:"ire"`
}

type CassandraExecuteCqlQueryRequest struct {
	Query       []byte      `thrift:"1,required" json:"query"`
	Compression Compression `thrift:"2,required" json:"compression"`
}

type CassandraExecuteCqlQueryResponse struct {
	Value *CqlResult                   `thrift:"0" json:"value"`
	Ire   *InvalidRequestException     `thrift:"1" json:"ire"`
	Ue    *UnavailableException        `thrift:"2" json:"ue"`
	Te    *TimedOutException           `thrift:"3" json:"te"`
	Sde   *SchemaDisagreementException `thrift:"4" json:"sde"`
}

type CassandraDescribeRingRequest struct {
	Keyspace string `thrift:"1,required" json:"keyspace"`
}

type CassandraDescribeRingResponse struct {
	Value []*TokenRange            `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
}

type CassandraRemoveRequest struct {
	Key              []byte           `thrift:"1,required" json:"key"`
	ColumnPath       *ColumnPath      `thrift:"2,required" json:"column_path"`
	Timestamp        int64            `thrift:"3,required" json:"timestamp"`
	ConsistencyLevel ConsistencyLevel `thrift:"4,required" json:"consistency_level"`
}

type CassandraRemoveResponse struct {
	Ire *InvalidRequestException `thrift:"1" json:"ire"`
	Ue  *UnavailableException    `thrift:"2" json:"ue"`
	Te  *TimedOutException       `thrift:"3" json:"te"`
}

type CassandraDescribeKeyspacesRequest struct {

}

type CassandraDescribeKeyspacesResponse struct {
	Value []*KsDef                 `thrift:"0" json:"value"`
	Ire   *InvalidRequestException `thrift:"1" json:"ire"`
}

type CassandraLoginRequest struct {
	AuthRequest *AuthenticationRequest `thrift:"1,required" json:"auth_request"`
}

type CassandraLoginResponse struct {
	Authnx *AuthenticationException `thrift:"1" json:"authnx"`
	Authzx *AuthorizationException  `thrift:"2" json:"authzx"`
}

type CassandraDescribeClusterNameRequest struct {

}

type CassandraDescribeClusterNameResponse struct {
	Value string `thrift:"0" json:"value"`
}

type RPCClient interface {
	Call(method string, request interface{}, response interface{}) error
}

type CassandraClient struct {
	Client RPCClient
}

func (s *CassandraClient) GetRangeSlices(ColumnParent *ColumnParent, Predicate *SlicePredicate, Range *KeyRange, ConsistencyLevel ConsistencyLevel) ([]*KeySlice, error) {
	req := &CassandraGetRangeSlicesRequest{
		ColumnParent:     ColumnParent,
		Predicate:        Predicate,
		Range:            Range,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraGetRangeSlicesResponse{}
	err := s.Client.Call("get_range_slices", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return res.Value, err
}

func (s *CassandraClient) SetKeyspace(Keyspace string) error {
	req := &CassandraSetKeyspaceRequest{
		Keyspace: Keyspace,
	}
	res := &CassandraSetKeyspaceResponse{}
	err := s.Client.Call("set_keyspace", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	}
	return err
}

func (s *CassandraClient) SystemDropKeyspace(Keyspace string) (string, error) {
	req := &CassandraSystemDropKeyspaceRequest{
		Keyspace: Keyspace,
	}
	res := &CassandraSystemDropKeyspaceResponse{}
	err := s.Client.Call("system_drop_keyspace", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Sde != nil:
		err = res.Sde
	}
	return res.Value, err
}

func (s *CassandraClient) SystemDropColumnFamily(ColumnFamily string) (string, error) {
	req := &CassandraSystemDropColumnFamilyRequest{
		ColumnFamily: ColumnFamily,
	}
	res := &CassandraSystemDropColumnFamilyResponse{}
	err := s.Client.Call("system_drop_column_family", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Sde != nil:
		err = res.Sde
	}
	return res.Value, err
}

func (s *CassandraClient) SystemAddColumnFamily(CfDef *CfDef) (string, error) {
	req := &CassandraSystemAddColumnFamilyRequest{
		CfDef: CfDef,
	}
	res := &CassandraSystemAddColumnFamilyResponse{}
	err := s.Client.Call("system_add_column_family", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Sde != nil:
		err = res.Sde
	}
	return res.Value, err
}

func (s *CassandraClient) DescribeSnitch() (string, error) {
	req := &CassandraDescribeSnitchRequest{}
	res := &CassandraDescribeSnitchResponse{}
	err := s.Client.Call("describe_snitch", req, res)
	return res.Value, err
}

func (s *CassandraClient) MultigetCount(Keys [][]byte, ColumnParent *ColumnParent, Predicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) (map[string]int32, error) {
	req := &CassandraMultigetCountRequest{
		Keys:             Keys,
		ColumnParent:     ColumnParent,
		Predicate:        Predicate,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraMultigetCountResponse{}
	err := s.Client.Call("multiget_count", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return res.Value, err
}

func (s *CassandraClient) DescribePartitioner() (string, error) {
	req := &CassandraDescribePartitionerRequest{}
	res := &CassandraDescribePartitionerResponse{}
	err := s.Client.Call("describe_partitioner", req, res)
	return res.Value, err
}

func (s *CassandraClient) DescribeSplits(CfName string, StartToken string, EndToken string, KeysPerSplit int32) ([]string, error) {
	req := &CassandraDescribeSplitsRequest{
		CfName:       CfName,
		StartToken:   StartToken,
		EndToken:     EndToken,
		KeysPerSplit: KeysPerSplit,
	}
	res := &CassandraDescribeSplitsResponse{}
	err := s.Client.Call("describe_splits", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	}
	return res.Value, err
}

func (s *CassandraClient) RemoveCounter(Key []byte, Path *ColumnPath, ConsistencyLevel ConsistencyLevel) error {
	req := &CassandraRemoveCounterRequest{
		Key:              Key,
		Path:             Path,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraRemoveCounterResponse{}
	err := s.Client.Call("remove_counter", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return err
}

func (s *CassandraClient) DescribeSchemaVersions() (map[string][]string, error) {
	req := &CassandraDescribeSchemaVersionsRequest{}
	res := &CassandraDescribeSchemaVersionsResponse{}
	err := s.Client.Call("describe_schema_versions", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	}
	return res.Value, err
}

func (s *CassandraClient) Add(Key []byte, ColumnParent *ColumnParent, Column *CounterColumn, ConsistencyLevel ConsistencyLevel) error {
	req := &CassandraAddRequest{
		Key:              Key,
		ColumnParent:     ColumnParent,
		Column:           Column,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraAddResponse{}
	err := s.Client.Call("add", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return err
}

func (s *CassandraClient) GetSlice(Key []byte, ColumnParent *ColumnParent, Predicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) ([]*ColumnOrSuperColumn, error) {
	req := &CassandraGetSliceRequest{
		Key:              Key,
		ColumnParent:     ColumnParent,
		Predicate:        Predicate,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraGetSliceResponse{}
	err := s.Client.Call("get_slice", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return res.Value, err
}

func (s *CassandraClient) GetCount(Key []byte, ColumnParent *ColumnParent, Predicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) (int32, error) {
	req := &CassandraGetCountRequest{
		Key:              Key,
		ColumnParent:     ColumnParent,
		Predicate:        Predicate,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraGetCountResponse{}
	err := s.Client.Call("get_count", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return res.Value, err
}

func (s *CassandraClient) DescribeVersion() (string, error) {
	req := &CassandraDescribeVersionRequest{}
	res := &CassandraDescribeVersionResponse{}
	err := s.Client.Call("describe_version", req, res)
	return res.Value, err
}

func (s *CassandraClient) MultigetSlice(Keys [][]byte, ColumnParent *ColumnParent, Predicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) (map[string][]*ColumnOrSuperColumn, error) {
	req := &CassandraMultigetSliceRequest{
		Keys:             Keys,
		ColumnParent:     ColumnParent,
		Predicate:        Predicate,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraMultigetSliceResponse{}
	err := s.Client.Call("multiget_slice", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return res.Value, err
}

func (s *CassandraClient) SystemUpdateColumnFamily(CfDef *CfDef) (string, error) {
	req := &CassandraSystemUpdateColumnFamilyRequest{
		CfDef: CfDef,
	}
	res := &CassandraSystemUpdateColumnFamilyResponse{}
	err := s.Client.Call("system_update_column_family", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Sde != nil:
		err = res.Sde
	}
	return res.Value, err
}

func (s *CassandraClient) Truncate(Cfname string) error {
	req := &CassandraTruncateRequest{
		Cfname: Cfname,
	}
	res := &CassandraTruncateResponse{}
	err := s.Client.Call("truncate", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	}
	return err
}

func (s *CassandraClient) Get(Key []byte, ColumnPath *ColumnPath, ConsistencyLevel ConsistencyLevel) (*ColumnOrSuperColumn, error) {
	req := &CassandraGetRequest{
		Key:              Key,
		ColumnPath:       ColumnPath,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraGetResponse{}
	err := s.Client.Call("get", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Nfe != nil:
		err = res.Nfe
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return res.Value, err
}

func (s *CassandraClient) BatchMutate(MutationMap map[string]map[string][]*Mutation, ConsistencyLevel ConsistencyLevel) error {
	req := &CassandraBatchMutateRequest{
		MutationMap:      MutationMap,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraBatchMutateResponse{}
	err := s.Client.Call("batch_mutate", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return err
}

func (s *CassandraClient) SystemUpdateKeyspace(KsDef *KsDef) (string, error) {
	req := &CassandraSystemUpdateKeyspaceRequest{
		KsDef: KsDef,
	}
	res := &CassandraSystemUpdateKeyspaceResponse{}
	err := s.Client.Call("system_update_keyspace", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Sde != nil:
		err = res.Sde
	}
	return res.Value, err
}

func (s *CassandraClient) SystemAddKeyspace(KsDef *KsDef) (string, error) {
	req := &CassandraSystemAddKeyspaceRequest{
		KsDef: KsDef,
	}
	res := &CassandraSystemAddKeyspaceResponse{}
	err := s.Client.Call("system_add_keyspace", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Sde != nil:
		err = res.Sde
	}
	return res.Value, err
}

func (s *CassandraClient) GetIndexedSlices(ColumnParent *ColumnParent, IndexClause *IndexClause, ColumnPredicate *SlicePredicate, ConsistencyLevel ConsistencyLevel) ([]*KeySlice, error) {
	req := &CassandraGetIndexedSlicesRequest{
		ColumnParent:     ColumnParent,
		IndexClause:      IndexClause,
		ColumnPredicate:  ColumnPredicate,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraGetIndexedSlicesResponse{}
	err := s.Client.Call("get_indexed_slices", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return res.Value, err
}

func (s *CassandraClient) Insert(Key []byte, ColumnParent *ColumnParent, Column *Column, ConsistencyLevel ConsistencyLevel) error {
	req := &CassandraInsertRequest{
		Key:              Key,
		ColumnParent:     ColumnParent,
		Column:           Column,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraInsertResponse{}
	err := s.Client.Call("insert", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return err
}

func (s *CassandraClient) DescribeKeyspace(Keyspace string) (*KsDef, error) {
	req := &CassandraDescribeKeyspaceRequest{
		Keyspace: Keyspace,
	}
	res := &CassandraDescribeKeyspaceResponse{}
	err := s.Client.Call("describe_keyspace", req, res)
	switch {
	case res.Nfe != nil:
		err = res.Nfe
	case res.Ire != nil:
		err = res.Ire
	}
	return res.Value, err
}

func (s *CassandraClient) ExecuteCqlQuery(Query []byte, Compression Compression) (*CqlResult, error) {
	req := &CassandraExecuteCqlQueryRequest{
		Query:       Query,
		Compression: Compression,
	}
	res := &CassandraExecuteCqlQueryResponse{}
	err := s.Client.Call("execute_cql_query", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	case res.Sde != nil:
		err = res.Sde
	}
	return res.Value, err
}

func (s *CassandraClient) DescribeRing(Keyspace string) ([]*TokenRange, error) {
	req := &CassandraDescribeRingRequest{
		Keyspace: Keyspace,
	}
	res := &CassandraDescribeRingResponse{}
	err := s.Client.Call("describe_ring", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	}
	return res.Value, err
}

func (s *CassandraClient) Remove(Key []byte, ColumnPath *ColumnPath, Timestamp int64, ConsistencyLevel ConsistencyLevel) error {
	req := &CassandraRemoveRequest{
		Key:              Key,
		ColumnPath:       ColumnPath,
		Timestamp:        Timestamp,
		ConsistencyLevel: ConsistencyLevel,
	}
	res := &CassandraRemoveResponse{}
	err := s.Client.Call("remove", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	case res.Ue != nil:
		err = res.Ue
	case res.Te != nil:
		err = res.Te
	}
	return err
}

func (s *CassandraClient) DescribeKeyspaces() ([]*KsDef, error) {
	req := &CassandraDescribeKeyspacesRequest{}
	res := &CassandraDescribeKeyspacesResponse{}
	err := s.Client.Call("describe_keyspaces", req, res)
	switch {
	case res.Ire != nil:
		err = res.Ire
	}
	return res.Value, err
}

func (s *CassandraClient) Login(AuthRequest *AuthenticationRequest) error {
	req := &CassandraLoginRequest{
		AuthRequest: AuthRequest,
	}
	res := &CassandraLoginResponse{}
	err := s.Client.Call("login", req, res)
	switch {
	case res.Authnx != nil:
		err = res.Authnx
	case res.Authzx != nil:
		err = res.Authzx
	}
	return err
}

func (s *CassandraClient) DescribeClusterName() (string, error) {
	req := &CassandraDescribeClusterNameRequest{}
	res := &CassandraDescribeClusterNameResponse{}
	err := s.Client.Call("describe_cluster_name", req, res)
	return res.Value, err
}

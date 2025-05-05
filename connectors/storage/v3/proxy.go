// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	stdlibtime "time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/redis/go-redis/v9"

	"github.com/ice-blockchain/wintr/log"
	"github.com/ice-blockchain/wintr/time"
)

func (l *lb) Pipeline() redis.Pipeliner {
	return l.instance().Pipeline()
}

func (l *lb) Pipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	return l.instance().Pipelined(ctx, fn) //nolint:wrapcheck // It's just a proxy.
}

func (l *lb) TxPipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error) {
	return l.instance().TxPipelined(ctx, fn) //nolint:wrapcheck // It's just a proxy.
}

func (l *lb) TxPipeline() redis.Pipeliner {
	return l.instance().TxPipeline()
}

func (l *lb) Command(ctx context.Context) *redis.CommandsInfoCmd {
	return l.instance().Command(ctx)
}

func (l *lb) CommandList(ctx context.Context, filter *redis.FilterBy) *redis.StringSliceCmd {
	return l.instance().CommandList(ctx, filter)
}

func (l *lb) CommandGetKeys(ctx context.Context, commands ...any) *redis.StringSliceCmd {
	return l.instance().CommandGetKeys(ctx, commands...)
}

func (l *lb) CommandGetKeysAndFlags(ctx context.Context, commands ...any) *redis.KeyFlagsCmd {
	return l.instance().CommandGetKeysAndFlags(ctx, commands...)
}

func (l *lb) ClientGetName(ctx context.Context) *redis.StringCmd {
	return l.instance().ClientGetName(ctx)
}

func (l *lb) Echo(ctx context.Context, message any) *redis.StringCmd {
	return l.instance().Echo(ctx, message)
}

func (l *lb) Quit(ctx context.Context) (res *redis.StatusCmd) {
	for _, inst := range l.instances {
		if elem := inst.Quit(ctx); res == nil || (res.Err() == nil && elem.Err() != nil) {
			res = elem
		}
	}

	return res
}

func (l *lb) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return l.instance().Del(ctx, keys...)
}

func (l *lb) Unlink(ctx context.Context, keys ...string) *redis.IntCmd {
	return l.instance().Unlink(ctx, keys...)
}

func (l *lb) Dump(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().Dump(ctx, key)
}

func (l *lb) Exists(ctx context.Context, keys ...string) *redis.IntCmd {
	return l.instance().Exists(ctx, keys...)
}

func (l *lb) Expire(ctx context.Context, key string, expiration stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().Expire(ctx, key, expiration)
}

func (l *lb) ExpireAt(ctx context.Context, key string, tm stdlibtime.Time) *redis.BoolCmd {
	return l.instance().ExpireAt(ctx, key, tm)
}

func (l *lb) ExpireTime(ctx context.Context, key string) *redis.DurationCmd {
	return l.instance().ExpireTime(ctx, key)
}

func (l *lb) ExpireNX(ctx context.Context, key string, expiration stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().ExpireNX(ctx, key, expiration)
}

func (l *lb) ExpireXX(ctx context.Context, key string, expiration stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().ExpireXX(ctx, key, expiration)
}

func (l *lb) ExpireGT(ctx context.Context, key string, expiration stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().ExpireGT(ctx, key, expiration)
}

func (l *lb) ExpireLT(ctx context.Context, key string, expiration stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().ExpireLT(ctx, key, expiration)
}

func (l *lb) Keys(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return l.instance().Keys(ctx, pattern)
}

//nolint:revive // We can't change the API.
func (l *lb) Migrate(ctx context.Context, host, port, key string, db int, timeout stdlibtime.Duration) *redis.StatusCmd {
	return l.instance().Migrate(ctx, host, port, key, db, timeout)
}

func (l *lb) Move(ctx context.Context, key string, db int) *redis.BoolCmd {
	return l.instance().Move(ctx, key, db)
}

func (l *lb) ObjectRefCount(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().ObjectRefCount(ctx, key)
}

func (l *lb) ObjectEncoding(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().ObjectEncoding(ctx, key)
}

func (l *lb) ObjectIdleTime(ctx context.Context, key string) *redis.DurationCmd {
	return l.instance().ObjectIdleTime(ctx, key)
}

func (l *lb) Persist(ctx context.Context, key string) *redis.BoolCmd {
	return l.instance().Persist(ctx, key)
}

func (l *lb) PExpire(ctx context.Context, key string, expiration stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().PExpire(ctx, key, expiration)
}

func (l *lb) PExpireAt(ctx context.Context, key string, tm stdlibtime.Time) *redis.BoolCmd {
	return l.instance().PExpireAt(ctx, key, tm)
}

func (l *lb) PExpireTime(ctx context.Context, key string) *redis.DurationCmd {
	return l.instance().PExpireTime(ctx, key)
}

func (l *lb) PTTL(ctx context.Context, key string) *redis.DurationCmd {
	return l.instance().PTTL(ctx, key)
}

func (l *lb) RandomKey(ctx context.Context) *redis.StringCmd {
	return l.instance().RandomKey(ctx)
}

func (l *lb) Rename(ctx context.Context, key, newkey string) *redis.StatusCmd {
	return l.instance().Rename(ctx, key, newkey)
}

func (l *lb) RenameNX(ctx context.Context, key, newkey string) *redis.BoolCmd {
	return l.instance().RenameNX(ctx, key, newkey)
}

func (l *lb) Restore(ctx context.Context, key string, ttl stdlibtime.Duration, value string) *redis.StatusCmd {
	return l.instance().Restore(ctx, key, ttl, value)
}

func (l *lb) RestoreReplace(ctx context.Context, key string, ttl stdlibtime.Duration, value string) *redis.StatusCmd {
	return l.instance().RestoreReplace(ctx, key, ttl, value)
}

func (l *lb) Sort(ctx context.Context, key string, sort *redis.Sort) *redis.StringSliceCmd {
	return l.instance().Sort(ctx, key, sort)
}

func (l *lb) SortRO(ctx context.Context, key string, sort *redis.Sort) *redis.StringSliceCmd {
	return l.instance().SortRO(ctx, key, sort)
}

func (l *lb) SortStore(ctx context.Context, key, store string, sort *redis.Sort) *redis.IntCmd {
	return l.instance().SortStore(ctx, key, store, sort)
}

func (l *lb) SortInterfaces(ctx context.Context, key string, sort *redis.Sort) *redis.SliceCmd {
	return l.instance().SortInterfaces(ctx, key, sort)
}

func (l *lb) Touch(ctx context.Context, keys ...string) *redis.IntCmd {
	return l.instance().Touch(ctx, keys...)
}

func (l *lb) TTL(ctx context.Context, key string) *redis.DurationCmd {
	return l.instance().TTL(ctx, key)
}

func (l *lb) Type(ctx context.Context, key string) *redis.StatusCmd {
	return l.instance().Type(ctx, key)
}

func (l *lb) Append(ctx context.Context, key, value string) *redis.IntCmd {
	return l.instance().Append(ctx, key, value)
}

func (l *lb) Decr(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().Decr(ctx, key)
}

func (l *lb) DecrBy(ctx context.Context, key string, decrement int64) *redis.IntCmd {
	return l.instance().DecrBy(ctx, key, decrement)
}

func (l *lb) Get(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().Get(ctx, key)
}

func (l *lb) GetRange(ctx context.Context, key string, start, end int64) *redis.StringCmd {
	return l.instance().GetRange(ctx, key, start, end)
}

func (l *lb) GetSet(ctx context.Context, key string, value any) *redis.StringCmd {
	return l.instance().GetSet(ctx, key, value)
}

func (l *lb) GetEx(ctx context.Context, key string, expiration stdlibtime.Duration) *redis.StringCmd {
	return l.instance().GetEx(ctx, key, expiration)
}

func (l *lb) GetDel(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().GetDel(ctx, key)
}

func (l *lb) Incr(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().Incr(ctx, key)
}

func (l *lb) IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd {
	return l.instance().IncrBy(ctx, key, value)
}

func (l *lb) IncrByFloat(ctx context.Context, key string, value float64) *redis.FloatCmd {
	return l.instance().IncrByFloat(ctx, key, value)
}

func (l *lb) MGet(ctx context.Context, keys ...string) *redis.SliceCmd {
	return l.instance().MGet(ctx, keys...)
}

func (l *lb) MSet(ctx context.Context, values ...any) *redis.StatusCmd {
	return l.instance().MSet(ctx, values...)
}

func (l *lb) MSetNX(ctx context.Context, values ...any) *redis.BoolCmd {
	return l.instance().MSetNX(ctx, values...)
}

func (l *lb) Set(ctx context.Context, key string, value any, expiration stdlibtime.Duration) *redis.StatusCmd {
	return l.instance().Set(ctx, key, value, expiration)
}

//nolint:gocritic // We can't change the API.
func (l *lb) SetArgs(ctx context.Context, key string, value any, a redis.SetArgs) *redis.StatusCmd {
	return l.instance().SetArgs(ctx, key, value, a)
}

func (l *lb) SetEx(ctx context.Context, key string, value any, expiration stdlibtime.Duration) *redis.StatusCmd {
	return l.instance().SetEx(ctx, key, value, expiration)
}

func (l *lb) SetNX(ctx context.Context, key string, value any, expiration stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().SetNX(ctx, key, value, expiration)
}

func (l *lb) SetXX(ctx context.Context, key string, value any, expiration stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().SetXX(ctx, key, value, expiration)
}

func (l *lb) SetRange(ctx context.Context, key string, offset int64, value string) *redis.IntCmd {
	return l.instance().SetRange(ctx, key, offset, value)
}

func (l *lb) StrLen(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().StrLen(ctx, key)
}

func (l *lb) Copy(ctx context.Context, sourceKey, destKey string, db int, replace bool) *redis.IntCmd {
	return l.instance().Copy(ctx, sourceKey, destKey, db, replace)
}

func (l *lb) GetBit(ctx context.Context, key string, offset int64) *redis.IntCmd {
	return l.instance().GetBit(ctx, key, offset)
}

func (l *lb) SetBit(ctx context.Context, key string, offset int64, value int) *redis.IntCmd {
	return l.instance().SetBit(ctx, key, offset, value)
}

func (l *lb) BitCount(ctx context.Context, key string, bitCount *redis.BitCount) *redis.IntCmd {
	return l.instance().BitCount(ctx, key, bitCount)
}

func (l *lb) BitOpAnd(ctx context.Context, destKey string, keys ...string) *redis.IntCmd {
	return l.instance().BitOpAnd(ctx, destKey, keys...)
}

func (l *lb) BitOpOr(ctx context.Context, destKey string, keys ...string) *redis.IntCmd {
	return l.instance().BitOpOr(ctx, destKey, keys...)
}

func (l *lb) BitOpXor(ctx context.Context, destKey string, keys ...string) *redis.IntCmd {
	return l.instance().BitOpXor(ctx, destKey, keys...)
}

func (l *lb) BitOpNot(ctx context.Context, destKey, key string) *redis.IntCmd {
	return l.instance().BitOpNot(ctx, destKey, key)
}

func (l *lb) BitPos(ctx context.Context, key string, bit int64, pos ...int64) *redis.IntCmd {
	return l.instance().BitPos(ctx, key, bit, pos...)
}

//nolint:revive // We can't change the API.
func (l *lb) BitPosSpan(ctx context.Context, key string, bit int8, start, end int64, span string) *redis.IntCmd {
	return l.instance().BitPosSpan(ctx, key, bit, start, end, span)
}

func (l *lb) BitField(ctx context.Context, key string, args ...any) *redis.IntSliceCmd {
	return l.instance().BitField(ctx, key, args...)
}

func (l *lb) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	return l.instance().Scan(ctx, cursor, match, count)
}

func (l *lb) ScanType(ctx context.Context, cursor uint64, match string, count int64, keyType string) *redis.ScanCmd {
	return l.instance().ScanType(ctx, cursor, match, count, keyType)
}

func (l *lb) SScan(ctx context.Context, key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return l.instance().SScan(ctx, key, cursor, match, count)
}

func (l *lb) HScan(ctx context.Context, key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return l.instance().HScan(ctx, key, cursor, match, count)
}

func (l *lb) HScanNoValues(ctx context.Context, key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return l.instance().HScanNoValues(ctx, key, cursor, match, count)
}

func (l *lb) ZScan(ctx context.Context, key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return l.instance().ZScan(ctx, key, cursor, match, count)
}

func (l *lb) HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd {
	return l.instance().HDel(ctx, key, fields...)
}

func (l *lb) HExists(ctx context.Context, key, field string) *redis.BoolCmd {
	return l.instance().HExists(ctx, key, field)
}

func (l *lb) HGet(ctx context.Context, key, field string) *redis.StringCmd {
	return l.instance().HGet(ctx, key, field)
}

func (l *lb) HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd {
	return l.instance().HGetAll(ctx, key)
}

func (l *lb) HIncrBy(ctx context.Context, key, field string, incr int64) *redis.IntCmd {
	return l.instance().HIncrBy(ctx, key, field, incr)
}

func (l *lb) HIncrByFloat(ctx context.Context, key, field string, incr float64) *redis.FloatCmd {
	return l.instance().HIncrByFloat(ctx, key, field, incr)
}

func (l *lb) HKeys(ctx context.Context, key string) *redis.StringSliceCmd {
	return l.instance().HKeys(ctx, key)
}

func (l *lb) HLen(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().HLen(ctx, key)
}

func (l *lb) HMGet(ctx context.Context, key string, fields ...string) *redis.SliceCmd {
	return l.instance().HMGet(ctx, key, fields...)
}

func (l *lb) HSet(ctx context.Context, key string, values ...any) *redis.IntCmd {
	return l.instance().HSet(ctx, key, values...)
}

func (l *lb) HMSet(ctx context.Context, key string, values ...any) *redis.BoolCmd {
	return l.instance().HMSet(ctx, key, values...)
}

func (l *lb) HSetNX(ctx context.Context, key, field string, value any) *redis.BoolCmd {
	return l.instance().HSetNX(ctx, key, field, value)
}

func (l *lb) HVals(ctx context.Context, key string) *redis.StringSliceCmd {
	return l.instance().HVals(ctx, key)
}

func (l *lb) HRandField(ctx context.Context, key string, count int) *redis.StringSliceCmd {
	return l.instance().HRandField(ctx, key, count)
}

func (l *lb) HRandFieldWithValues(ctx context.Context, key string, count int) *redis.KeyValueSliceCmd {
	return l.instance().HRandFieldWithValues(ctx, key, count)
}

func (l *lb) BLPop(ctx context.Context, timeout stdlibtime.Duration, keys ...string) *redis.StringSliceCmd {
	return l.instance().BLPop(ctx, timeout, keys...)
}

func (l *lb) BLMPop(ctx context.Context, timeout stdlibtime.Duration, direction string, count int64, keys ...string) *redis.KeyValuesCmd {
	return l.instance().BLMPop(ctx, timeout, direction, count, keys...)
}

func (l *lb) BRPop(ctx context.Context, timeout stdlibtime.Duration, keys ...string) *redis.StringSliceCmd {
	return l.instance().BRPop(ctx, timeout, keys...)
}

func (l *lb) BRPopLPush(ctx context.Context, source, destination string, timeout stdlibtime.Duration) *redis.StringCmd {
	return l.instance().BRPopLPush(ctx, source, destination, timeout)
}

func (l *lb) LCS(ctx context.Context, q *redis.LCSQuery) *redis.LCSCmd {
	return l.instance().LCS(ctx, q)
}

func (l *lb) LIndex(ctx context.Context, key string, index int64) *redis.StringCmd {
	return l.instance().LIndex(ctx, key, index)
}

func (l *lb) LInsert(ctx context.Context, key, op string, pivot, value any) *redis.IntCmd {
	return l.instance().LInsert(ctx, key, op, pivot, value)
}

func (l *lb) LInsertBefore(ctx context.Context, key string, pivot, value any) *redis.IntCmd {
	return l.instance().LInsertBefore(ctx, key, pivot, value)
}

func (l *lb) LInsertAfter(ctx context.Context, key string, pivot, value any) *redis.IntCmd {
	return l.instance().LInsertAfter(ctx, key, pivot, value)
}

func (l *lb) LLen(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().LLen(ctx, key)
}

func (l *lb) LMPop(ctx context.Context, direction string, count int64, keys ...string) *redis.KeyValuesCmd {
	return l.instance().LMPop(ctx, direction, count, keys...)
}

func (l *lb) LPop(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().LPop(ctx, key)
}

func (l *lb) LPopCount(ctx context.Context, key string, count int) *redis.StringSliceCmd {
	return l.instance().LPopCount(ctx, key, count)
}

func (l *lb) LPos(ctx context.Context, key, value string, args redis.LPosArgs) *redis.IntCmd {
	return l.instance().LPos(ctx, key, value, args)
}

func (l *lb) LPosCount(ctx context.Context, key, value string, count int64, args redis.LPosArgs) *redis.IntSliceCmd {
	return l.instance().LPosCount(ctx, key, value, count, args)
}

func (l *lb) LPush(ctx context.Context, key string, values ...any) *redis.IntCmd {
	return l.instance().LPush(ctx, key, values...)
}

func (l *lb) LPushX(ctx context.Context, key string, values ...any) *redis.IntCmd {
	return l.instance().LPushX(ctx, key, values...)
}

func (l *lb) LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return l.instance().LRange(ctx, key, start, stop)
}

func (l *lb) LRem(ctx context.Context, key string, count int64, value any) *redis.IntCmd {
	return l.instance().LRem(ctx, key, count, value)
}

func (l *lb) LSet(ctx context.Context, key string, index int64, value any) *redis.StatusCmd {
	return l.instance().LSet(ctx, key, index, value)
}

func (l *lb) LTrim(ctx context.Context, key string, start, stop int64) *redis.StatusCmd {
	return l.instance().LTrim(ctx, key, start, stop)
}

func (l *lb) RPop(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().RPop(ctx, key)
}

func (l *lb) RPopCount(ctx context.Context, key string, count int) *redis.StringSliceCmd {
	return l.instance().RPopCount(ctx, key, count)
}

func (l *lb) RPopLPush(ctx context.Context, source, destination string) *redis.StringCmd {
	return l.instance().RPopLPush(ctx, source, destination)
}

func (l *lb) RPush(ctx context.Context, key string, values ...any) *redis.IntCmd {
	return l.instance().RPush(ctx, key, values...)
}

func (l *lb) RPushX(ctx context.Context, key string, values ...any) *redis.IntCmd {
	return l.instance().RPushX(ctx, key, values...)
}

func (l *lb) LMove(ctx context.Context, source, destination, srcpos, destpos string) *redis.StringCmd {
	return l.instance().LMove(ctx, source, destination, srcpos, destpos)
}

//nolint:revive // We can't change the API.
func (l *lb) BLMove(ctx context.Context, source, destination, srcpos, destpos string, timeout stdlibtime.Duration) *redis.StringCmd {
	return l.instance().BLMove(ctx, source, destination, srcpos, destpos, timeout)
}

func (l *lb) SAdd(ctx context.Context, key string, members ...any) *redis.IntCmd {
	return l.instance().SAdd(ctx, key, members...)
}

func (l *lb) SCard(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().SCard(ctx, key)
}

func (l *lb) SDiff(ctx context.Context, keys ...string) *redis.StringSliceCmd {
	return l.instance().SDiff(ctx, keys...)
}

func (l *lb) SDiffStore(ctx context.Context, destination string, keys ...string) *redis.IntCmd {
	return l.instance().SDiffStore(ctx, destination, keys...)
}

func (l *lb) SInter(ctx context.Context, keys ...string) *redis.StringSliceCmd {
	return l.instance().SInter(ctx, keys...)
}

func (l *lb) SInterCard(ctx context.Context, limit int64, keys ...string) *redis.IntCmd {
	return l.instance().SInterCard(ctx, limit, keys...)
}

func (l *lb) SInterStore(ctx context.Context, destination string, keys ...string) *redis.IntCmd {
	return l.instance().SInterStore(ctx, destination, keys...)
}

func (l *lb) SIsMember(ctx context.Context, key string, member any) *redis.BoolCmd {
	return l.instance().SIsMember(ctx, key, member)
}

func (l *lb) SMIsMember(ctx context.Context, key string, members ...any) *redis.BoolSliceCmd {
	return l.instance().SMIsMember(ctx, key, members...)
}

func (l *lb) SMembers(ctx context.Context, key string) *redis.StringSliceCmd {
	return l.instance().SMembers(ctx, key)
}

func (l *lb) SMembersMap(ctx context.Context, key string) *redis.StringStructMapCmd {
	return l.instance().SMembersMap(ctx, key)
}

func (l *lb) SMove(ctx context.Context, source, destination string, member any) *redis.BoolCmd {
	return l.instance().SMove(ctx, source, destination, member)
}

func (l *lb) SPop(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().SPop(ctx, key)
}

func (l *lb) SPopN(ctx context.Context, key string, count int64) *redis.StringSliceCmd {
	return l.instance().SPopN(ctx, key, count)
}

func (l *lb) SRandMember(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().SRandMember(ctx, key)
}

func (l *lb) SRandMemberN(ctx context.Context, key string, count int64) *redis.StringSliceCmd {
	return l.instance().SRandMemberN(ctx, key, count)
}

func (l *lb) SRem(ctx context.Context, key string, members ...any) *redis.IntCmd {
	return l.instance().SRem(ctx, key, members...)
}

func (l *lb) SUnion(ctx context.Context, keys ...string) *redis.StringSliceCmd {
	return l.instance().SUnion(ctx, keys...)
}

func (l *lb) SUnionStore(ctx context.Context, destination string, keys ...string) *redis.IntCmd {
	return l.instance().SUnionStore(ctx, destination, keys...)
}

func (l *lb) XAdd(ctx context.Context, a *redis.XAddArgs) *redis.StringCmd {
	return l.instance().XAdd(ctx, a)
}

func (l *lb) XDel(ctx context.Context, stream string, ids ...string) *redis.IntCmd {
	return l.instance().XDel(ctx, stream, ids...)
}

func (l *lb) XLen(ctx context.Context, stream string) *redis.IntCmd {
	return l.instance().XLen(ctx, stream)
}

func (l *lb) XRange(ctx context.Context, stream, start, stop string) *redis.XMessageSliceCmd {
	return l.instance().XRange(ctx, stream, start, stop)
}

func (l *lb) XRangeN(ctx context.Context, stream, start, stop string, count int64) *redis.XMessageSliceCmd {
	return l.instance().XRangeN(ctx, stream, start, stop, count)
}

func (l *lb) XRevRange(ctx context.Context, stream, start, stop string) *redis.XMessageSliceCmd {
	return l.instance().XRevRange(ctx, stream, start, stop)
}

func (l *lb) XRevRangeN(ctx context.Context, stream, start, stop string, count int64) *redis.XMessageSliceCmd {
	return l.instance().XRevRangeN(ctx, stream, start, stop, count)
}

func (l *lb) XRead(ctx context.Context, a *redis.XReadArgs) *redis.XStreamSliceCmd {
	return l.instance().XRead(ctx, a)
}

func (l *lb) XReadStreams(ctx context.Context, streams ...string) *redis.XStreamSliceCmd {
	return l.instance().XReadStreams(ctx, streams...)
}

func (l *lb) XGroupCreate(ctx context.Context, stream, group, start string) *redis.StatusCmd {
	return l.instance().XGroupCreate(ctx, stream, group, start)
}

func (l *lb) XGroupCreateMkStream(ctx context.Context, stream, group, start string) *redis.StatusCmd {
	return l.instance().XGroupCreateMkStream(ctx, stream, group, start)
}

func (l *lb) XGroupSetID(ctx context.Context, stream, group, start string) *redis.StatusCmd {
	return l.instance().XGroupSetID(ctx, stream, group, start)
}

func (l *lb) XGroupDestroy(ctx context.Context, stream, group string) *redis.IntCmd {
	return l.instance().XGroupDestroy(ctx, stream, group)
}

func (l *lb) XGroupCreateConsumer(ctx context.Context, stream, group, consumer string) *redis.IntCmd {
	return l.instance().XGroupCreateConsumer(ctx, stream, group, consumer)
}

func (l *lb) XGroupDelConsumer(ctx context.Context, stream, group, consumer string) *redis.IntCmd {
	return l.instance().XGroupDelConsumer(ctx, stream, group, consumer)
}

func (l *lb) XReadGroup(ctx context.Context, a *redis.XReadGroupArgs) *redis.XStreamSliceCmd {
	return l.instance().XReadGroup(ctx, a)
}

func (l *lb) XAck(ctx context.Context, stream, group string, ids ...string) *redis.IntCmd {
	return l.instance().XAck(ctx, stream, group, ids...)
}

func (l *lb) XPending(ctx context.Context, stream, group string) *redis.XPendingCmd {
	return l.instance().XPending(ctx, stream, group)
}

func (l *lb) XPendingExt(ctx context.Context, a *redis.XPendingExtArgs) *redis.XPendingExtCmd {
	return l.instance().XPendingExt(ctx, a)
}

func (l *lb) XClaim(ctx context.Context, a *redis.XClaimArgs) *redis.XMessageSliceCmd {
	return l.instance().XClaim(ctx, a)
}

func (l *lb) XClaimJustID(ctx context.Context, a *redis.XClaimArgs) *redis.StringSliceCmd {
	return l.instance().XClaimJustID(ctx, a)
}

func (l *lb) XAutoClaim(ctx context.Context, a *redis.XAutoClaimArgs) *redis.XAutoClaimCmd {
	return l.instance().XAutoClaim(ctx, a)
}

func (l *lb) XAutoClaimJustID(ctx context.Context, a *redis.XAutoClaimArgs) *redis.XAutoClaimJustIDCmd {
	return l.instance().XAutoClaimJustID(ctx, a)
}

func (l *lb) XTrimMaxLen(ctx context.Context, key string, maximumLen int64) *redis.IntCmd {
	return l.instance().XTrimMaxLen(ctx, key, maximumLen)
}

func (l *lb) XTrimMaxLenApprox(ctx context.Context, key string, maximumLen, limit int64) *redis.IntCmd {
	return l.instance().XTrimMaxLenApprox(ctx, key, maximumLen, limit)
}

func (l *lb) XTrimMinID(ctx context.Context, key, minID string) *redis.IntCmd {
	return l.instance().XTrimMinID(ctx, key, minID)
}

func (l *lb) XTrimMinIDApprox(ctx context.Context, key, minID string, limit int64) *redis.IntCmd {
	return l.instance().XTrimMinIDApprox(ctx, key, minID, limit)
}

func (l *lb) XInfoGroups(ctx context.Context, key string) *redis.XInfoGroupsCmd {
	return l.instance().XInfoGroups(ctx, key)
}

func (l *lb) XInfoStream(ctx context.Context, key string) *redis.XInfoStreamCmd {
	return l.instance().XInfoStream(ctx, key)
}

func (l *lb) XInfoStreamFull(ctx context.Context, key string, count int) *redis.XInfoStreamFullCmd {
	return l.instance().XInfoStreamFull(ctx, key, count)
}

func (l *lb) XInfoConsumers(ctx context.Context, key, group string) *redis.XInfoConsumersCmd {
	return l.instance().XInfoConsumers(ctx, key, group)
}

func (l *lb) BZPopMax(ctx context.Context, timeout stdlibtime.Duration, keys ...string) *redis.ZWithKeyCmd {
	return l.instance().BZPopMax(ctx, timeout, keys...)
}

func (l *lb) BZPopMin(ctx context.Context, timeout stdlibtime.Duration, keys ...string) *redis.ZWithKeyCmd {
	return l.instance().BZPopMin(ctx, timeout, keys...)
}

func (l *lb) BZMPop(ctx context.Context, timeout stdlibtime.Duration, order string, count int64, keys ...string) *redis.ZSliceWithKeyCmd {
	return l.instance().BZMPop(ctx, timeout, order, count, keys...)
}

func (l *lb) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return l.instance().ZAdd(ctx, key, members...)
}

func (l *lb) ZAddLT(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return l.instance().ZAddLT(ctx, key, members...)
}

func (l *lb) ZAddGT(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return l.instance().ZAddGT(ctx, key, members...)
}

func (l *lb) ZAddNX(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return l.instance().ZAddNX(ctx, key, members...)
}

func (l *lb) ZAddXX(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	return l.instance().ZAddXX(ctx, key, members...)
}

//nolint:gocritic // We can't change the API.
func (l *lb) ZAddArgs(ctx context.Context, key string, args redis.ZAddArgs) *redis.IntCmd {
	return l.instance().ZAddArgs(ctx, key, args)
}

//nolint:gocritic // We can't change the API.
func (l *lb) ZAddArgsIncr(ctx context.Context, key string, args redis.ZAddArgs) *redis.FloatCmd {
	return l.instance().ZAddArgsIncr(ctx, key, args)
}

func (l *lb) ZCard(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().ZCard(ctx, key)
}

func (l *lb) ZCount(ctx context.Context, key, minimum, maximum string) *redis.IntCmd {
	return l.instance().ZCount(ctx, key, minimum, maximum)
}

func (l *lb) ZLexCount(ctx context.Context, key, minimum, maximum string) *redis.IntCmd {
	return l.instance().ZLexCount(ctx, key, minimum, maximum)
}

func (l *lb) ZIncrBy(ctx context.Context, key string, increment float64, member string) *redis.FloatCmd {
	return l.instance().ZIncrBy(ctx, key, increment, member)
}

func (l *lb) ZInter(ctx context.Context, store *redis.ZStore) *redis.StringSliceCmd {
	return l.instance().ZInter(ctx, store)
}

func (l *lb) ZInterWithScores(ctx context.Context, store *redis.ZStore) *redis.ZSliceCmd {
	return l.instance().ZInterWithScores(ctx, store)
}

func (l *lb) ZInterCard(ctx context.Context, limit int64, keys ...string) *redis.IntCmd {
	return l.instance().ZInterCard(ctx, limit, keys...)
}

func (l *lb) ZInterStore(ctx context.Context, destination string, store *redis.ZStore) *redis.IntCmd {
	return l.instance().ZInterStore(ctx, destination, store)
}

func (l *lb) ZMPop(ctx context.Context, order string, count int64, keys ...string) *redis.ZSliceWithKeyCmd {
	return l.instance().ZMPop(ctx, order, count, keys...)
}

func (l *lb) ZMScore(ctx context.Context, key string, members ...string) *redis.FloatSliceCmd {
	return l.instance().ZMScore(ctx, key, members...)
}

func (l *lb) ZPopMax(ctx context.Context, key string, count ...int64) *redis.ZSliceCmd {
	return l.instance().ZPopMax(ctx, key, count...)
}

func (l *lb) ZPopMin(ctx context.Context, key string, count ...int64) *redis.ZSliceCmd {
	return l.instance().ZPopMin(ctx, key, count...)
}

func (l *lb) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return l.instance().ZRange(ctx, key, start, stop)
}

func (l *lb) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
	return l.instance().ZRangeWithScores(ctx, key, start, stop)
}

func (l *lb) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return l.instance().ZRangeByScore(ctx, key, opt)
}

func (l *lb) ZRangeByLex(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return l.instance().ZRangeByLex(ctx, key, opt)
}

func (l *lb) ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	return l.instance().ZRangeByScoreWithScores(ctx, key, opt)
}

//nolint:gocritic // We can't change the API.
func (l *lb) ZRangeArgs(ctx context.Context, z redis.ZRangeArgs) *redis.StringSliceCmd {
	return l.instance().ZRangeArgs(ctx, z)
}

//nolint:gocritic // We can't change the API.
func (l *lb) ZRangeArgsWithScores(ctx context.Context, z redis.ZRangeArgs) *redis.ZSliceCmd {
	return l.instance().ZRangeArgsWithScores(ctx, z)
}

//nolint:gocritic // We can't change the API.
func (l *lb) ZRangeStore(ctx context.Context, dst string, z redis.ZRangeArgs) *redis.IntCmd {
	return l.instance().ZRangeStore(ctx, dst, z)
}

func (l *lb) ZRank(ctx context.Context, key, member string) *redis.IntCmd {
	return l.instance().ZRank(ctx, key, member)
}

func (l *lb) ZRankWithScore(ctx context.Context, key, member string) *redis.RankWithScoreCmd {
	return l.instance().ZRankWithScore(ctx, key, member)
}

func (l *lb) ZRem(ctx context.Context, key string, members ...any) *redis.IntCmd {
	return l.instance().ZRem(ctx, key, members...)
}

func (l *lb) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) *redis.IntCmd {
	return l.instance().ZRemRangeByRank(ctx, key, start, stop)
}

func (l *lb) ZRemRangeByScore(ctx context.Context, key, minimum, maximum string) *redis.IntCmd {
	return l.instance().ZRemRangeByScore(ctx, key, minimum, maximum)
}

func (l *lb) ZRemRangeByLex(ctx context.Context, key, minimum, maximum string) *redis.IntCmd {
	return l.instance().ZRemRangeByLex(ctx, key, minimum, maximum)
}

func (l *lb) ZRevRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	return l.instance().ZRevRange(ctx, key, start, stop)
}

func (l *lb) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
	return l.instance().ZRevRangeWithScores(ctx, key, start, stop)
}

func (l *lb) ZRevRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return l.instance().ZRevRangeByScore(ctx, key, opt)
}

func (l *lb) ZRevRangeByLex(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return l.instance().ZRevRangeByLex(ctx, key, opt)
}

func (l *lb) ZRevRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	return l.instance().ZRevRangeByScoreWithScores(ctx, key, opt)
}

func (l *lb) ZRevRank(ctx context.Context, key, member string) *redis.IntCmd {
	return l.instance().ZRevRank(ctx, key, member)
}

func (l *lb) ZRevRankWithScore(ctx context.Context, key, member string) *redis.RankWithScoreCmd {
	return l.instance().ZRevRankWithScore(ctx, key, member)
}

func (l *lb) ZScore(ctx context.Context, key, member string) *redis.FloatCmd {
	return l.instance().ZScore(ctx, key, member)
}

func (l *lb) ZUnionStore(ctx context.Context, dest string, store *redis.ZStore) *redis.IntCmd {
	return l.instance().ZUnionStore(ctx, dest, store)
}

func (l *lb) ZRandMember(ctx context.Context, key string, count int) *redis.StringSliceCmd {
	return l.instance().ZRandMember(ctx, key, count)
}

func (l *lb) ZRandMemberWithScores(ctx context.Context, key string, count int) *redis.ZSliceCmd {
	return l.instance().ZRandMemberWithScores(ctx, key, count)
}

//nolint:gocritic // We can't change the API.
func (l *lb) ZUnion(ctx context.Context, store redis.ZStore) *redis.StringSliceCmd {
	return l.instance().ZUnion(ctx, store)
}

//nolint:gocritic // We can't change the API.
func (l *lb) ZUnionWithScores(ctx context.Context, store redis.ZStore) *redis.ZSliceCmd {
	return l.instance().ZUnionWithScores(ctx, store)
}

func (l *lb) ZDiff(ctx context.Context, keys ...string) *redis.StringSliceCmd {
	return l.instance().ZDiff(ctx, keys...)
}

func (l *lb) ZDiffWithScores(ctx context.Context, keys ...string) *redis.ZSliceCmd {
	return l.instance().ZDiffWithScores(ctx, keys...)
}

func (l *lb) ZDiffStore(ctx context.Context, destination string, keys ...string) *redis.IntCmd {
	return l.instance().ZDiffStore(ctx, destination, keys...)
}

func (l *lb) PFAdd(ctx context.Context, key string, els ...any) *redis.IntCmd {
	return l.instance().PFAdd(ctx, key, els...)
}

func (l *lb) PFCount(ctx context.Context, keys ...string) *redis.IntCmd {
	return l.instance().PFCount(ctx, keys...)
}

func (l *lb) PFMerge(ctx context.Context, dest string, keys ...string) *redis.StatusCmd {
	return l.instance().PFMerge(ctx, dest, keys...)
}

func (l *lb) BgRewriteAOF(ctx context.Context) *redis.StatusCmd {
	return l.instance().BgRewriteAOF(ctx)
}

func (l *lb) BgSave(ctx context.Context) *redis.StatusCmd {
	return l.instance().BgSave(ctx)
}

func (l *lb) ClientKill(ctx context.Context, ipPort string) *redis.StatusCmd {
	return l.instance().ClientKill(ctx, ipPort)
}

func (l *lb) ClientKillByFilter(ctx context.Context, keys ...string) *redis.IntCmd {
	return l.instance().ClientKillByFilter(ctx, keys...)
}

func (l *lb) ClientList(ctx context.Context) *redis.StringCmd {
	return l.instance().ClientList(ctx)
}

func (l *lb) ClientInfo(ctx context.Context) *redis.ClientInfoCmd {
	return l.instance().ClientInfo(ctx)
}

func (l *lb) ClientPause(ctx context.Context, dur stdlibtime.Duration) *redis.BoolCmd {
	return l.instance().ClientPause(ctx, dur)
}

func (l *lb) ClientUnpause(ctx context.Context) *redis.BoolCmd {
	return l.instance().ClientUnpause(ctx)
}

func (l *lb) ClientID(ctx context.Context) *redis.IntCmd {
	return l.instance().ClientID(ctx)
}

func (l *lb) ClientUnblock(ctx context.Context, id int64) *redis.IntCmd {
	return l.instance().ClientUnblock(ctx, id)
}

func (l *lb) ClientUnblockWithError(ctx context.Context, id int64) *redis.IntCmd {
	return l.instance().ClientUnblockWithError(ctx, id)
}

func (l *lb) ConfigGet(ctx context.Context, parameter string) *redis.MapStringStringCmd {
	return l.instance().ConfigGet(ctx, parameter)
}

func (l *lb) ConfigResetStat(ctx context.Context) *redis.StatusCmd {
	return l.instance().ConfigResetStat(ctx)
}

func (l *lb) ConfigSet(ctx context.Context, parameter, value string) *redis.StatusCmd {
	return l.instance().ConfigSet(ctx, parameter, value)
}

func (l *lb) ConfigRewrite(ctx context.Context) *redis.StatusCmd {
	return l.instance().ConfigRewrite(ctx)
}

func (l *lb) DBSize(ctx context.Context) *redis.IntCmd {
	return l.instance().DBSize(ctx)
}

func (l *lb) FlushAll(ctx context.Context) *redis.StatusCmd {
	return l.instance().FlushAll(ctx)
}

func (l *lb) FlushAllAsync(ctx context.Context) *redis.StatusCmd {
	return l.instance().FlushAllAsync(ctx)
}

func (l *lb) FlushDB(ctx context.Context) *redis.StatusCmd {
	return l.instance().FlushDB(ctx)
}

func (l *lb) FlushDBAsync(ctx context.Context) *redis.StatusCmd {
	return l.instance().FlushDBAsync(ctx)
}

func (l *lb) Info(ctx context.Context, section ...string) *redis.StringCmd {
	return l.instance().Info(ctx, section...)
}

func (l *lb) LastSave(ctx context.Context) *redis.IntCmd {
	return l.instance().LastSave(ctx)
}

func (l *lb) Save(ctx context.Context) *redis.StatusCmd {
	return l.instance().Save(ctx)
}

func (l *lb) Shutdown(ctx context.Context) *redis.StatusCmd {
	return l.instance().Shutdown(ctx)
}

func (l *lb) ShutdownSave(ctx context.Context) *redis.StatusCmd {
	return l.instance().ShutdownSave(ctx)
}

func (l *lb) ShutdownNoSave(ctx context.Context) *redis.StatusCmd {
	return l.instance().ShutdownNoSave(ctx)
}

func (l *lb) SlaveOf(ctx context.Context, host, port string) *redis.StatusCmd {
	return l.instance().SlaveOf(ctx, host, port)
}

func (l *lb) SlowLogGet(ctx context.Context, num int64) *redis.SlowLogCmd {
	return l.instance().SlowLogGet(ctx, num)
}

func (l *lb) Time(ctx context.Context) *redis.TimeCmd {
	return l.instance().Time(ctx)
}

func (l *lb) DebugObject(ctx context.Context, key string) *redis.StringCmd {
	return l.instance().DebugObject(ctx, key)
}

func (l *lb) ReadOnly(ctx context.Context) *redis.StatusCmd {
	return l.instance().ReadOnly(ctx)
}

func (l *lb) ReadWrite(ctx context.Context) *redis.StatusCmd {
	return l.instance().ReadWrite(ctx)
}

func (l *lb) MemoryUsage(ctx context.Context, key string, samples ...int) *redis.IntCmd {
	return l.instance().MemoryUsage(ctx, key, samples...)
}

func (l *lb) Eval(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd {
	return l.instance().Eval(ctx, script, keys, args...)
}

func (l *lb) EvalSha(ctx context.Context, sha1 string, keys []string, args ...any) *redis.Cmd {
	return l.instance().EvalSha(ctx, sha1, keys, args...)
}

func (l *lb) EvalRO(ctx context.Context, script string, keys []string, args ...any) *redis.Cmd {
	return l.instance().EvalRO(ctx, script, keys, args...)
}

func (l *lb) EvalShaRO(ctx context.Context, sha1 string, keys []string, args ...any) *redis.Cmd {
	return l.instance().EvalShaRO(ctx, sha1, keys, args...)
}

func (l *lb) ScriptExists(ctx context.Context, hashes ...string) *redis.BoolSliceCmd {
	return l.instance().ScriptExists(ctx, hashes...)
}

func (l *lb) ScriptFlush(ctx context.Context) *redis.StatusCmd {
	return l.instance().ScriptFlush(ctx)
}

func (l *lb) ScriptKill(ctx context.Context) *redis.StatusCmd {
	return l.instance().ScriptKill(ctx)
}

func (l *lb) ScriptLoad(ctx context.Context, script string) *redis.StringCmd {
	return l.instance().ScriptLoad(ctx, script)
}

func (l *lb) FunctionLoad(ctx context.Context, code string) *redis.StringCmd {
	return l.instance().FunctionLoad(ctx, code)
}

func (l *lb) FunctionLoadReplace(ctx context.Context, code string) *redis.StringCmd {
	return l.instance().FunctionLoadReplace(ctx, code)
}

func (l *lb) FunctionDelete(ctx context.Context, libName string) *redis.StringCmd {
	return l.instance().FunctionDelete(ctx, libName)
}

func (l *lb) FunctionFlush(ctx context.Context) *redis.StringCmd {
	return l.instance().FunctionFlush(ctx)
}

func (l *lb) FunctionKill(ctx context.Context) *redis.StringCmd {
	return l.instance().FunctionKill(ctx)
}

func (l *lb) FunctionFlushAsync(ctx context.Context) *redis.StringCmd {
	return l.instance().FunctionFlushAsync(ctx)
}

func (l *lb) FunctionList(ctx context.Context, q redis.FunctionListQuery) *redis.FunctionListCmd {
	return l.instance().FunctionList(ctx, q)
}

func (l *lb) FunctionDump(ctx context.Context) *redis.StringCmd {
	return l.instance().FunctionDump(ctx)
}

func (l *lb) FunctionRestore(ctx context.Context, libDump string) *redis.StringCmd {
	return l.instance().FunctionRestore(ctx, libDump)
}

func (l *lb) FunctionStats(ctx context.Context) *redis.FunctionStatsCmd {
	return l.instance().FunctionStats(ctx)
}

func (l *lb) FCall(ctx context.Context, function string, keys []string, args ...any) *redis.Cmd {
	return l.instance().FCall(ctx, function, keys, args...)
}

func (l *lb) FCallRo(ctx context.Context, function string, keys []string, args ...any) *redis.Cmd {
	return l.instance().FCallRo(ctx, function, keys, args...)
}

//nolint:revive // Its part of the API.
func (l *lb) FCallRO(ctx context.Context, function string, keys []string, args ...any) *redis.Cmd {
	return l.instance().FCallRO(ctx, function, keys, args...)
}

func (l *lb) Publish(ctx context.Context, channel string, message any) *redis.IntCmd {
	return l.instance().Publish(ctx, channel, message)
}

func (l *lb) SPublish(ctx context.Context, channel string, message any) *redis.IntCmd {
	return l.instance().SPublish(ctx, channel, message)
}

func (l *lb) PubSubChannels(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return l.instance().PubSubChannels(ctx, pattern)
}

func (l *lb) PubSubNumSub(ctx context.Context, channels ...string) *redis.MapStringIntCmd {
	return l.instance().PubSubNumSub(ctx, channels...)
}

func (l *lb) PubSubNumPat(ctx context.Context) *redis.IntCmd {
	return l.instance().PubSubNumPat(ctx)
}

func (l *lb) PubSubShardChannels(ctx context.Context, pattern string) *redis.StringSliceCmd {
	return l.instance().PubSubShardChannels(ctx, pattern)
}

func (l *lb) PubSubShardNumSub(ctx context.Context, channels ...string) *redis.MapStringIntCmd {
	return l.instance().PubSubShardNumSub(ctx, channels...)
}

func (l *lb) ClusterMyShardID(ctx context.Context) *redis.StringCmd {
	return l.instance().ClusterMyShardID(ctx)
}

func (l *lb) ClusterSlots(ctx context.Context) *redis.ClusterSlotsCmd {
	return l.instance().ClusterSlots(ctx)
}

func (l *lb) ClusterShards(ctx context.Context) *redis.ClusterShardsCmd {
	return l.instance().ClusterShards(ctx)
}

func (l *lb) ClusterLinks(ctx context.Context) *redis.ClusterLinksCmd {
	return l.instance().ClusterLinks(ctx)
}

func (l *lb) ClusterNodes(ctx context.Context) *redis.StringCmd {
	return l.instance().ClusterNodes(ctx)
}

func (l *lb) ClusterMeet(ctx context.Context, host, port string) *redis.StatusCmd {
	return l.instance().ClusterMeet(ctx, host, port)
}

func (l *lb) ClusterForget(ctx context.Context, nodeID string) *redis.StatusCmd {
	return l.instance().ClusterForget(ctx, nodeID)
}

func (l *lb) ClusterReplicate(ctx context.Context, nodeID string) *redis.StatusCmd {
	return l.instance().ClusterReplicate(ctx, nodeID)
}

func (l *lb) ClusterResetSoft(ctx context.Context) *redis.StatusCmd {
	return l.instance().ClusterResetSoft(ctx)
}

func (l *lb) ClusterResetHard(ctx context.Context) *redis.StatusCmd {
	return l.instance().ClusterResetHard(ctx)
}

func (l *lb) ClusterInfo(ctx context.Context) *redis.StringCmd {
	return l.instance().ClusterInfo(ctx)
}

func (l *lb) ClusterKeySlot(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().ClusterKeySlot(ctx, key)
}

func (l *lb) ClusterGetKeysInSlot(ctx context.Context, slot, count int) *redis.StringSliceCmd {
	return l.instance().ClusterGetKeysInSlot(ctx, slot, count)
}

func (l *lb) ClusterCountFailureReports(ctx context.Context, nodeID string) *redis.IntCmd {
	return l.instance().ClusterCountFailureReports(ctx, nodeID)
}

func (l *lb) ClusterCountKeysInSlot(ctx context.Context, slot int) *redis.IntCmd {
	return l.instance().ClusterCountKeysInSlot(ctx, slot)
}

func (l *lb) ClusterDelSlots(ctx context.Context, slots ...int) *redis.StatusCmd {
	return l.instance().ClusterDelSlots(ctx, slots...)
}

func (l *lb) ClusterDelSlotsRange(ctx context.Context, minimum, maximum int) *redis.StatusCmd {
	return l.instance().ClusterDelSlotsRange(ctx, minimum, maximum)
}

func (l *lb) ClusterSaveConfig(ctx context.Context) *redis.StatusCmd {
	return l.instance().ClusterSaveConfig(ctx)
}

func (l *lb) ClusterSlaves(ctx context.Context, nodeID string) *redis.StringSliceCmd {
	return l.instance().ClusterSlaves(ctx, nodeID)
}

func (l *lb) ClusterFailover(ctx context.Context) *redis.StatusCmd {
	return l.instance().ClusterFailover(ctx)
}

func (l *lb) ClusterAddSlots(ctx context.Context, slots ...int) *redis.StatusCmd {
	return l.instance().ClusterAddSlots(ctx, slots...)
}

func (l *lb) ClusterAddSlotsRange(ctx context.Context, mininum, maximum int) *redis.StatusCmd {
	return l.instance().ClusterAddSlotsRange(ctx, mininum, maximum)
}

func (l *lb) GeoAdd(ctx context.Context, key string, geoLocation ...*redis.GeoLocation) *redis.IntCmd {
	return l.instance().GeoAdd(ctx, key, geoLocation...)
}

func (l *lb) GeoPos(ctx context.Context, key string, members ...string) *redis.GeoPosCmd {
	return l.instance().GeoPos(ctx, key, members...)
}

func (l *lb) GeoRadius(ctx context.Context, key string, longitude, latitude float64, query *redis.GeoRadiusQuery) *redis.GeoLocationCmd {
	return l.instance().GeoRadius(ctx, key, longitude, latitude, query)
}

func (l *lb) GeoRadiusStore(ctx context.Context, key string, longitude, latitude float64, query *redis.GeoRadiusQuery) *redis.IntCmd {
	return l.instance().GeoRadiusStore(ctx, key, longitude, latitude, query)
}

func (l *lb) GeoRadiusByMember(ctx context.Context, key, member string, query *redis.GeoRadiusQuery) *redis.GeoLocationCmd {
	return l.instance().GeoRadiusByMember(ctx, key, member, query)
}

func (l *lb) GeoRadiusByMemberStore(ctx context.Context, key, member string, query *redis.GeoRadiusQuery) *redis.IntCmd {
	return l.instance().GeoRadiusByMemberStore(ctx, key, member, query)
}

func (l *lb) GeoSearch(ctx context.Context, key string, q *redis.GeoSearchQuery) *redis.StringSliceCmd {
	return l.instance().GeoSearch(ctx, key, q)
}

func (l *lb) GeoSearchLocation(ctx context.Context, key string, q *redis.GeoSearchLocationQuery) *redis.GeoSearchLocationCmd {
	return l.instance().GeoSearchLocation(ctx, key, q)
}

func (l *lb) GeoSearchStore(ctx context.Context, key, store string, q *redis.GeoSearchStoreQuery) *redis.IntCmd {
	return l.instance().GeoSearchStore(ctx, key, store, q)
}

func (l *lb) GeoDist(ctx context.Context, key, member1, member2, unit string) *redis.FloatCmd {
	return l.instance().GeoDist(ctx, key, member1, member2, unit)
}

func (l *lb) GeoHash(ctx context.Context, key string, members ...string) *redis.StringSliceCmd {
	return l.instance().GeoHash(ctx, key, members...)
}

func (l *lb) ACLDryRun(ctx context.Context, username string, command ...any) *redis.StringCmd {
	return l.instance().ACLDryRun(ctx, username, command...)
}

func (l *lb) ACLLog(ctx context.Context, count int64) *redis.ACLLogCmd {
	return l.instance().ACLLog(ctx, count)
}

func (l *lb) ACLLogReset(ctx context.Context) *redis.StatusCmd {
	return l.instance().ACLLogReset(ctx)
}

func (l *lb) ModuleLoadex(ctx context.Context, conf *redis.ModuleLoadexConfig) *redis.StringCmd {
	return l.instance().ModuleLoadex(ctx, conf)
}

func (l *lb) BFAdd(ctx context.Context, key string, element any) *redis.BoolCmd {
	return l.instance().BFAdd(ctx, key, element)
}

func (l *lb) BFCard(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().BFCard(ctx, key)
}

func (l *lb) BFExists(ctx context.Context, key string, element any) *redis.BoolCmd {
	return l.instance().BFExists(ctx, key, element)
}

func (l *lb) BFInfo(ctx context.Context, key string) *redis.BFInfoCmd {
	return l.instance().BFInfo(ctx, key)
}

func (l *lb) BFInfoArg(ctx context.Context, key, option string) *redis.BFInfoCmd {
	return l.instance().BFInfoArg(ctx, key, option)
}

func (l *lb) BFInfoCapacity(ctx context.Context, key string) *redis.BFInfoCmd {
	return l.instance().BFInfoCapacity(ctx, key)
}

func (l *lb) BFInfoSize(ctx context.Context, key string) *redis.BFInfoCmd {
	return l.instance().BFInfoSize(ctx, key)
}

func (l *lb) BFInfoFilters(ctx context.Context, key string) *redis.BFInfoCmd {
	return l.instance().BFInfoFilters(ctx, key)
}

func (l *lb) BFInfoItems(ctx context.Context, key string) *redis.BFInfoCmd {
	return l.instance().BFInfoItems(ctx, key)
}

func (l *lb) BFInfoExpansion(ctx context.Context, key string) *redis.BFInfoCmd {
	return l.instance().BFInfoExpansion(ctx, key)
}

func (l *lb) BFInsert(ctx context.Context, key string, options *redis.BFInsertOptions, elements ...any) *redis.BoolSliceCmd {
	return l.instance().BFInsert(ctx, key, options, elements...)
}

func (l *lb) BFMAdd(ctx context.Context, key string, elements ...any) *redis.BoolSliceCmd {
	return l.instance().BFMAdd(ctx, key, elements...)
}

func (l *lb) BFMExists(ctx context.Context, key string, elements ...any) *redis.BoolSliceCmd {
	return l.instance().BFMExists(ctx, key, elements...)
}

func (l *lb) BFReserve(ctx context.Context, key string, errorRate float64, capacity int64) *redis.StatusCmd {
	return l.instance().BFReserve(ctx, key, errorRate, capacity)
}

func (l *lb) BFReserveExpansion(ctx context.Context, key string, errorRate float64, capacity, expansion int64) *redis.StatusCmd {
	return l.instance().BFReserveExpansion(ctx, key, errorRate, capacity, expansion)
}

func (l *lb) BFReserveNonScaling(ctx context.Context, key string, errorRate float64, capacity int64) *redis.StatusCmd {
	return l.instance().BFReserveNonScaling(ctx, key, errorRate, capacity)
}

func (l *lb) BFReserveWithArgs(ctx context.Context, key string, options *redis.BFReserveOptions) *redis.StatusCmd {
	return l.instance().BFReserveWithArgs(ctx, key, options)
}

func (l *lb) BFScanDump(ctx context.Context, key string, iterator int64) *redis.ScanDumpCmd {
	return l.instance().BFScanDump(ctx, key, iterator)
}

func (l *lb) BFLoadChunk(ctx context.Context, key string, iterator int64, data any) *redis.StatusCmd {
	return l.instance().BFLoadChunk(ctx, key, iterator, data)
}

func (l *lb) CFAdd(ctx context.Context, key string, element any) *redis.BoolCmd {
	return l.instance().CFAdd(ctx, key, element)
}

func (l *lb) CFAddNX(ctx context.Context, key string, element any) *redis.BoolCmd {
	return l.instance().CFAddNX(ctx, key, element)
}

func (l *lb) CFCount(ctx context.Context, key string, element any) *redis.IntCmd {
	return l.instance().CFCount(ctx, key, element)
}

func (l *lb) CFDel(ctx context.Context, key string, element any) *redis.BoolCmd {
	return l.instance().CFDel(ctx, key, element)
}

func (l *lb) CFExists(ctx context.Context, key string, element any) *redis.BoolCmd {
	return l.instance().CFExists(ctx, key, element)
}

func (l *lb) CFInfo(ctx context.Context, key string) *redis.CFInfoCmd {
	return l.instance().CFInfo(ctx, key)
}

func (l *lb) CFInsert(ctx context.Context, key string, options *redis.CFInsertOptions, elements ...any) *redis.BoolSliceCmd {
	return l.instance().CFInsert(ctx, key, options, elements...)
}

func (l *lb) CFInsertNX(ctx context.Context, key string, options *redis.CFInsertOptions, elements ...any) *redis.IntSliceCmd {
	return l.instance().CFInsertNX(ctx, key, options, elements...)
}

func (l *lb) CFMExists(ctx context.Context, key string, elements ...any) *redis.BoolSliceCmd {
	return l.instance().CFMExists(ctx, key, elements...)
}

func (l *lb) CFReserve(ctx context.Context, key string, capacity int64) *redis.StatusCmd {
	return l.instance().CFReserve(ctx, key, capacity)
}

func (l *lb) CFReserveWithArgs(ctx context.Context, key string, options *redis.CFReserveOptions) *redis.StatusCmd {
	return l.instance().CFReserveWithArgs(ctx, key, options)
}

func (l *lb) CFReserveExpansion(ctx context.Context, key string, capacity, expansion int64) *redis.StatusCmd {
	return l.instance().CFReserveExpansion(ctx, key, capacity, expansion)
}

func (l *lb) CFReserveBucketSize(ctx context.Context, key string, capacity, bucketsize int64) *redis.StatusCmd {
	return l.instance().CFReserveBucketSize(ctx, key, capacity, bucketsize)
}

func (l *lb) CFReserveMaxIterations(ctx context.Context, key string, capacity, maximumiterations int64) *redis.StatusCmd {
	return l.instance().CFReserveMaxIterations(ctx, key, capacity, maximumiterations)
}

func (l *lb) CFScanDump(ctx context.Context, key string, iterator int64) *redis.ScanDumpCmd {
	return l.instance().CFScanDump(ctx, key, iterator)
}

func (l *lb) CFLoadChunk(ctx context.Context, key string, iterator int64, data any) *redis.StatusCmd {
	return l.instance().CFLoadChunk(ctx, key, iterator, data)
}

func (l *lb) CMSIncrBy(ctx context.Context, key string, elements ...any) *redis.IntSliceCmd {
	return l.instance().CMSIncrBy(ctx, key, elements...)
}

func (l *lb) CMSInfo(ctx context.Context, key string) *redis.CMSInfoCmd {
	return l.instance().CMSInfo(ctx, key)
}

func (l *lb) CMSInitByDim(ctx context.Context, key string, width, height int64) *redis.StatusCmd {
	return l.instance().CMSInitByDim(ctx, key, width, height)
}

func (l *lb) CMSInitByProb(ctx context.Context, key string, errorRate, probability float64) *redis.StatusCmd {
	return l.instance().CMSInitByProb(ctx, key, errorRate, probability)
}

func (l *lb) CMSMerge(ctx context.Context, destKey string, sourceKeys ...string) *redis.StatusCmd {
	return l.instance().CMSMerge(ctx, destKey, sourceKeys...)
}

func (l *lb) CMSMergeWithWeight(ctx context.Context, destKey string, sourceKeys map[string]int64) *redis.StatusCmd {
	return l.instance().CMSMergeWithWeight(ctx, destKey, sourceKeys)
}

func (l *lb) CMSQuery(ctx context.Context, key string, elements ...any) *redis.IntSliceCmd {
	return l.instance().CMSQuery(ctx, key, elements...)
}

func (l *lb) TopKAdd(ctx context.Context, key string, elements ...any) *redis.StringSliceCmd {
	return l.instance().TopKAdd(ctx, key, elements...)
}

func (l *lb) TopKCount(ctx context.Context, key string, elements ...any) *redis.IntSliceCmd {
	return l.instance().TopKCount(ctx, key, elements...)
}

func (l *lb) TopKIncrBy(ctx context.Context, key string, elements ...any) *redis.StringSliceCmd {
	return l.instance().TopKIncrBy(ctx, key, elements...)
}

func (l *lb) TopKInfo(ctx context.Context, key string) *redis.TopKInfoCmd {
	return l.instance().TopKInfo(ctx, key)
}

func (l *lb) TopKList(ctx context.Context, key string) *redis.StringSliceCmd {
	return l.instance().TopKList(ctx, key)
}

func (l *lb) TopKListWithCount(ctx context.Context, key string) *redis.MapStringIntCmd {
	return l.instance().TopKListWithCount(ctx, key)
}

func (l *lb) TopKQuery(ctx context.Context, key string, elements ...any) *redis.BoolSliceCmd {
	return l.instance().TopKQuery(ctx, key, elements...)
}

func (l *lb) TopKReserve(ctx context.Context, key string, k int64) *redis.StatusCmd {
	return l.instance().TopKReserve(ctx, key, k)
}

//nolint:revive // API.
func (l *lb) TopKReserveWithOptions(ctx context.Context, key string, k, width, depth int64, decay float64) *redis.StatusCmd {
	return l.instance().TopKReserveWithOptions(ctx, key, k, width, depth, decay)
}

func (l *lb) TDigestAdd(ctx context.Context, key string, elements ...float64) *redis.StatusCmd {
	return l.instance().TDigestAdd(ctx, key, elements...)
}

func (l *lb) TDigestByRank(ctx context.Context, key string, rank ...uint64) *redis.FloatSliceCmd {
	return l.instance().TDigestByRank(ctx, key, rank...)
}

func (l *lb) TDigestByRevRank(ctx context.Context, key string, rank ...uint64) *redis.FloatSliceCmd {
	return l.instance().TDigestByRevRank(ctx, key, rank...)
}

func (l *lb) TDigestCDF(ctx context.Context, key string, elements ...float64) *redis.FloatSliceCmd {
	return l.instance().TDigestCDF(ctx, key, elements...)
}

func (l *lb) TDigestCreate(ctx context.Context, key string) *redis.StatusCmd {
	return l.instance().TDigestCreate(ctx, key)
}

func (l *lb) TDigestCreateWithCompression(ctx context.Context, key string, compression int64) *redis.StatusCmd {
	return l.instance().TDigestCreateWithCompression(ctx, key, compression)
}

func (l *lb) TDigestInfo(ctx context.Context, key string) *redis.TDigestInfoCmd {
	return l.instance().TDigestInfo(ctx, key)
}

func (l *lb) TDigestMax(ctx context.Context, key string) *redis.FloatCmd {
	return l.instance().TDigestMax(ctx, key)
}

func (l *lb) TDigestMin(ctx context.Context, key string) *redis.FloatCmd {
	return l.instance().TDigestMin(ctx, key)
}

func (l *lb) TDigestMerge(ctx context.Context, destKey string, options *redis.TDigestMergeOptions, sourceKeys ...string) *redis.StatusCmd {
	return l.instance().TDigestMerge(ctx, destKey, options, sourceKeys...)
}

func (l *lb) TDigestQuantile(ctx context.Context, key string, elements ...float64) *redis.FloatSliceCmd {
	return l.instance().TDigestQuantile(ctx, key, elements...)
}

func (l *lb) TDigestRank(ctx context.Context, key string, values ...float64) *redis.IntSliceCmd {
	return l.instance().TDigestRank(ctx, key, values...)
}

func (l *lb) TDigestReset(ctx context.Context, key string) *redis.StatusCmd {
	return l.instance().TDigestReset(ctx, key)
}

func (l *lb) TDigestRevRank(ctx context.Context, key string, values ...float64) *redis.IntSliceCmd {
	return l.instance().TDigestRevRank(ctx, key, values...)
}

func (l *lb) TDigestTrimmedMean(ctx context.Context, key string, lowCutQuantile, highCutQuantile float64) *redis.FloatCmd {
	return l.instance().TDigestTrimmedMean(ctx, key, lowCutQuantile, highCutQuantile)
}

func (l *lb) JSONArrAppend(ctx context.Context, key, path string, values ...any) *redis.IntSliceCmd {
	return l.instance().JSONArrAppend(ctx, key, path, values...)
}

func (l *lb) JSONArrIndex(ctx context.Context, key, path string, values ...any) *redis.IntSliceCmd {
	return l.instance().JSONArrIndex(ctx, key, path, values...)
}

func (l *lb) JSONArrInsert(ctx context.Context, key, path string, index int64, values ...any) *redis.IntSliceCmd {
	return l.instance().JSONArrInsert(ctx, key, path, index, values...)
}

func (l *lb) JSONArrLen(ctx context.Context, key, path string) *redis.IntSliceCmd {
	return l.instance().JSONArrLen(ctx, key, path)
}

func (l *lb) JSONGet(ctx context.Context, key string, paths ...string) *redis.JSONCmd {
	return l.instance().JSONGet(ctx, key, paths...)
}

func (l *lb) JSONSet(ctx context.Context, key, path string, value any) *redis.StatusCmd {
	return l.instance().JSONSet(ctx, key, path, value)
}

func (l *lb) JSONDel(ctx context.Context, key, path string) *redis.IntCmd {
	return l.instance().JSONDel(ctx, key, path)
}

func (l *lb) JSONMGet(ctx context.Context, path string, keys ...string) *redis.JSONSliceCmd {
	return l.instance().JSONMGet(ctx, path, keys...)
}

func (l *lb) JSONMSet(ctx context.Context, values ...any) *redis.StatusCmd {
	return l.instance().JSONMSet(ctx, values...)
}

func (l *lb) JSONArrIndexWithArgs(ctx context.Context, key, path string, options *redis.JSONArrIndexArgs, value ...any) *redis.IntSliceCmd {
	return l.instance().JSONArrIndexWithArgs(ctx, key, path, options, value...)
}

func (l *lb) JSONArrPop(ctx context.Context, key, path string, index int) *redis.StringSliceCmd {
	return l.instance().JSONArrPop(ctx, key, path, index)
}

func (l *lb) JSONArrTrim(ctx context.Context, key, path string) *redis.IntSliceCmd {
	return l.instance().JSONArrTrim(ctx, key, path)
}

func (l *lb) JSONArrTrimWithArgs(ctx context.Context, key, path string, options *redis.JSONArrTrimArgs) *redis.IntSliceCmd {
	return l.instance().JSONArrTrimWithArgs(ctx, key, path, options)
}

func (l *lb) JSONClear(ctx context.Context, key, path string) *redis.IntCmd {
	return l.instance().JSONClear(ctx, key, path)
}

func (l *lb) JSONDebugMemory(ctx context.Context, key, path string) *redis.IntCmd {
	return l.instance().JSONDebugMemory(ctx, key, path)
}

func (l *lb) JSONForget(ctx context.Context, key, path string) *redis.IntCmd {
	return l.instance().JSONForget(ctx, key, path)
}

func (l *lb) JSONGetWithArgs(ctx context.Context, key string, options *redis.JSONGetArgs, paths ...string) *redis.JSONCmd {
	return l.instance().JSONGetWithArgs(ctx, key, options, paths...)
}

func (l *lb) JSONMerge(ctx context.Context, key, path, value string) *redis.StatusCmd {
	return l.instance().JSONMerge(ctx, key, path, value)
}

func (l *lb) JSONMSetArgs(ctx context.Context, docs []redis.JSONSetArgs) *redis.StatusCmd {
	return l.instance().JSONMSetArgs(ctx, docs)
}

func (l *lb) JSONNumIncrBy(ctx context.Context, key, path string, value float64) *redis.JSONCmd {
	return l.instance().JSONNumIncrBy(ctx, key, path, value)
}

func (l *lb) JSONObjKeys(ctx context.Context, key, path string) *redis.SliceCmd {
	return l.instance().JSONObjKeys(ctx, key, path)
}

func (l *lb) JSONObjLen(ctx context.Context, key, path string) *redis.IntPointerSliceCmd {
	return l.instance().JSONObjLen(ctx, key, path)
}

func (l *lb) JSONSetMode(ctx context.Context, key, path string, value any, mode string) *redis.StatusCmd {
	return l.instance().JSONSetMode(ctx, key, path, value, mode)
}

func (l *lb) JSONStrAppend(ctx context.Context, key, path, value string) *redis.IntPointerSliceCmd {
	return l.instance().JSONStrAppend(ctx, key, path, value)
}

func (l *lb) JSONStrLen(ctx context.Context, key, path string) *redis.IntPointerSliceCmd {
	return l.instance().JSONStrLen(ctx, key, path)
}

func (l *lb) JSONToggle(ctx context.Context, key, path string) *redis.IntPointerSliceCmd {
	return l.instance().JSONToggle(ctx, key, path)
}

func (l *lb) JSONType(ctx context.Context, key, path string) *redis.JSONSliceCmd {
	return l.instance().JSONType(ctx, key, path)
}

func (l *lb) TSAdd(ctx context.Context, key string, timestamp any, value float64) *redis.IntCmd {
	return l.instance().TSAdd(ctx, key, timestamp, value)
}

func (l *lb) TSAddWithArgs(ctx context.Context, key string, timestamp any, value float64, options *redis.TSOptions) *redis.IntCmd {
	return l.instance().TSAddWithArgs(ctx, key, timestamp, value, options)
}

func (l *lb) TSCreate(ctx context.Context, key string) *redis.StatusCmd {
	return l.instance().TSCreate(ctx, key)
}

func (l *lb) TSCreateWithArgs(ctx context.Context, key string, options *redis.TSOptions) *redis.StatusCmd {
	return l.instance().TSCreateWithArgs(ctx, key, options)
}

func (l *lb) TSAlter(ctx context.Context, key string, options *redis.TSAlterOptions) *redis.StatusCmd {
	return l.instance().TSAlter(ctx, key, options)
}

func (l *lb) TSCreateRule(ctx context.Context, sourceKey, destKey string, aggregator redis.Aggregator, bucketDuration int) *redis.StatusCmd {
	return l.instance().TSCreateRule(ctx, sourceKey, destKey, aggregator, bucketDuration)
}

//nolint:lll,revive // API.
func (l *lb) TSCreateRuleWithArgs(ctx context.Context, sourceKey, destKey string, aggregator redis.Aggregator, bucketDuration int, options *redis.TSCreateRuleOptions) *redis.StatusCmd {
	return l.instance().TSCreateRuleWithArgs(ctx, sourceKey, destKey, aggregator, bucketDuration, options)
}

func (l *lb) TSIncrBy(ctx context.Context, key string, timestamp float64) *redis.IntCmd {
	return l.instance().TSIncrBy(ctx, key, timestamp)
}

func (l *lb) TSIncrByWithArgs(ctx context.Context, key string, timestamp float64, options *redis.TSIncrDecrOptions) *redis.IntCmd {
	return l.instance().TSIncrByWithArgs(ctx, key, timestamp, options)
}

func (l *lb) TSDecrBy(ctx context.Context, key string, timestamp float64) *redis.IntCmd {
	return l.instance().TSDecrBy(ctx, key, timestamp)
}

func (l *lb) TSDecrByWithArgs(ctx context.Context, key string, timestamp float64, options *redis.TSIncrDecrOptions) *redis.IntCmd {
	return l.instance().TSDecrByWithArgs(ctx, key, timestamp, options)
}

func (l *lb) TSDel(ctx context.Context, key string, fromTimestamp, toTimestamp int) *redis.IntCmd {
	return l.instance().TSDel(ctx, key, fromTimestamp, toTimestamp)
}

func (l *lb) TSDeleteRule(ctx context.Context, sourceKey, destKey string) *redis.StatusCmd {
	return l.instance().TSDeleteRule(ctx, sourceKey, destKey)
}

func (l *lb) TSGet(ctx context.Context, key string) *redis.TSTimestampValueCmd {
	return l.instance().TSGet(ctx, key)
}

func (l *lb) TSGetWithArgs(ctx context.Context, key string, options *redis.TSGetOptions) *redis.TSTimestampValueCmd {
	return l.instance().TSGetWithArgs(ctx, key, options)
}

func (l *lb) TSInfo(ctx context.Context, key string) *redis.MapStringInterfaceCmd {
	return l.instance().TSInfo(ctx, key)
}

func (l *lb) TSInfoWithArgs(ctx context.Context, key string, options *redis.TSInfoOptions) *redis.MapStringInterfaceCmd {
	return l.instance().TSInfoWithArgs(ctx, key, options)
}

func (l *lb) TSMAdd(ctx context.Context, ktvSlices [][]any) *redis.IntSliceCmd {
	return l.instance().TSMAdd(ctx, ktvSlices)
}

func (l *lb) TSQueryIndex(ctx context.Context, filterExpr []string) *redis.StringSliceCmd {
	return l.instance().TSQueryIndex(ctx, filterExpr)
}

func (l *lb) TSRevRange(ctx context.Context, key string, fromTimestamp, toTimestamp int) *redis.TSTimestampValueSliceCmd {
	return l.instance().TSRevRange(ctx, key, fromTimestamp, toTimestamp)
}

//nolint:lll // .
func (l *lb) TSRevRangeWithArgs(ctx context.Context, key string, fromTimestamp, toTimestamp int, options *redis.TSRevRangeOptions) *redis.TSTimestampValueSliceCmd {
	return l.instance().TSRevRangeWithArgs(ctx, key, fromTimestamp, toTimestamp, options)
}

func (l *lb) TSRange(ctx context.Context, key string, fromTimestamp, toTimestamp int) *redis.TSTimestampValueSliceCmd {
	return l.instance().TSRange(ctx, key, fromTimestamp, toTimestamp)
}

func (l *lb) TSRangeWithArgs(ctx context.Context, key string, fromTimestamp, toTimestamp int, options *redis.TSRangeOptions) *redis.TSTimestampValueSliceCmd {
	return l.instance().TSRangeWithArgs(ctx, key, fromTimestamp, toTimestamp, options)
}

func (l *lb) TSMRange(ctx context.Context, fromTimestamp, toTimestamp int, filterExpr []string) *redis.MapStringSliceInterfaceCmd {
	return l.instance().TSMRange(ctx, fromTimestamp, toTimestamp, filterExpr)
}

//nolint:lll // .
func (l *lb) TSMRangeWithArgs(ctx context.Context, fromTimestamp, toTimestamp int, filterExpr []string, options *redis.TSMRangeOptions) *redis.MapStringSliceInterfaceCmd {
	return l.instance().TSMRangeWithArgs(ctx, fromTimestamp, toTimestamp, filterExpr, options)
}

func (l *lb) TSMRevRange(ctx context.Context, fromTimestamp, toTimestamp int, filterExpr []string) *redis.MapStringSliceInterfaceCmd {
	return l.instance().TSMRevRange(ctx, fromTimestamp, toTimestamp, filterExpr)
}

//nolint:lll // .
func (l *lb) TSMRevRangeWithArgs(ctx context.Context, fromTimestamp, toTimestamp int, filterExpr []string, options *redis.TSMRevRangeOptions) *redis.MapStringSliceInterfaceCmd {
	return l.instance().TSMRevRangeWithArgs(ctx, fromTimestamp, toTimestamp, filterExpr, options)
}

func (l *lb) TSMGet(ctx context.Context, filters []string) *redis.MapStringSliceInterfaceCmd {
	return l.instance().TSMGet(ctx, filters)
}

func (l *lb) TSMGetWithArgs(ctx context.Context, filters []string, options *redis.TSMGetOptions) *redis.MapStringSliceInterfaceCmd {
	return l.instance().TSMGetWithArgs(ctx, filters, options)
}

func (l *lb) ObjectFreq(ctx context.Context, key string) *redis.IntCmd {
	return l.instance().ObjectFreq(ctx, key)
}

func (l *lb) BitFieldRO(ctx context.Context, key string, values ...interface{}) *redis.IntSliceCmd { //nolint:revive // Comes from interface.
	return l.instance().BitFieldRO(ctx, key, values...)
}

func (l *lb) HExpire(ctx context.Context, key string, expiration stdlibtime.Duration, fields ...string) *redis.IntSliceCmd {
	return l.instance().HExpire(ctx, key, expiration, fields...)
}

func (l *lb) HExpireWithArgs(
	ctx context.Context, key string, expiration stdlibtime.Duration, expirationArgs redis.HExpireArgs, fields ...string,
) *redis.IntSliceCmd {
	return l.instance().HExpireWithArgs(ctx, key, expiration, expirationArgs, fields...)
}

func (l *lb) HPExpire(ctx context.Context, key string, expiration stdlibtime.Duration, fields ...string) *redis.IntSliceCmd {
	return l.instance().HPExpire(ctx, key, expiration, fields...)
}

func (l *lb) HPExpireWithArgs(
	ctx context.Context, key string, expiration stdlibtime.Duration, expirationArgs redis.HExpireArgs, fields ...string,
) *redis.IntSliceCmd {
	return l.instance().HPExpireWithArgs(ctx, key, expiration, expirationArgs, fields...)
}

func (l *lb) HExpireAt(ctx context.Context, key string, tm stdlibtime.Time, fields ...string) *redis.IntSliceCmd {
	return l.instance().HExpireAt(ctx, key, tm, fields...)
}

func (l *lb) HExpireAtWithArgs(ctx context.Context, key string, tm stdlibtime.Time, expirationArgs redis.HExpireArgs, fields ...string) *redis.IntSliceCmd {
	return l.instance().HExpireAtWithArgs(ctx, key, tm, expirationArgs, fields...)
}

func (l *lb) HPExpireAt(ctx context.Context, key string, tm stdlibtime.Time, fields ...string) *redis.IntSliceCmd {
	return l.instance().HPExpireAt(ctx, key, tm, fields...)
}

func (l *lb) HPExpireAtWithArgs(ctx context.Context, key string, tm stdlibtime.Time, expirationArgs redis.HExpireArgs, fields ...string) *redis.IntSliceCmd {
	return l.instance().HPExpireAtWithArgs(ctx, key, tm, expirationArgs, fields...)
}

func (l *lb) HPersist(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return l.instance().HPersist(ctx, key, fields...)
}

func (l *lb) HExpireTime(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return l.instance().HExpireTime(ctx, key, fields...)
}

func (l *lb) HPExpireTime(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return l.instance().HPExpireTime(ctx, key, fields...)
}

func (l *lb) HTTL(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return l.instance().HTTL(ctx, key, fields...)
}

func (l *lb) HPTTL(ctx context.Context, key string, fields ...string) *redis.IntSliceCmd {
	return l.instance().HPTTL(ctx, key, fields...)
}

func (l *lb) FT_List(ctx context.Context) *redis.StringSliceCmd { //nolint:revive,stylecheck // Go-redis interface.
	return l.instance().FT_List(ctx)
}

func (l *lb) FTAggregate(ctx context.Context, index, query string) *redis.MapStringInterfaceCmd {
	return l.instance().FTAggregate(ctx, index, query)
}

func (l *lb) FTAggregateWithArgs(ctx context.Context, index, query string, options *redis.FTAggregateOptions) *redis.AggregateCmd {
	return l.instance().FTAggregateWithArgs(ctx, index, query, options)
}

func (l *lb) FTAliasAdd(ctx context.Context, index, alias string) *redis.StatusCmd {
	return l.instance().FTAliasAdd(ctx, index, alias)
}

func (l *lb) FTAliasDel(ctx context.Context, alias string) *redis.StatusCmd {
	return l.instance().FTAliasDel(ctx, alias)
}

func (l *lb) FTAliasUpdate(ctx context.Context, index, alias string) *redis.StatusCmd {
	return l.instance().FTAliasUpdate(ctx, index, alias)
}

func (l *lb) FTAlter(ctx context.Context, index string, skipInitialScan bool, definition []any) *redis.StatusCmd {
	return l.instance().FTAlter(ctx, index, skipInitialScan, definition)
}

func (l *lb) FTConfigGet(ctx context.Context, option string) *redis.MapMapStringInterfaceCmd { //nolint:staticcheck // .
	return l.instance().FTConfigGet(ctx, option)
}

func (l *lb) FTConfigSet(ctx context.Context, option string, value any) *redis.StatusCmd { //nolint:staticcheck // .
	return l.instance().FTConfigSet(ctx, option, value)
}

func (l *lb) FTCreate(ctx context.Context, index string, options *redis.FTCreateOptions, schema ...*redis.FieldSchema) *redis.StatusCmd {
	return l.instance().FTCreate(ctx, index, options, schema...)
}

func (l *lb) FTCursorDel(ctx context.Context, index string, cursorId int) *redis.StatusCmd { //nolint:stylecheck // .
	return l.instance().FTCursorDel(ctx, index, cursorId)
}

func (l *lb) FTCursorRead(ctx context.Context, index string, cursorId, count int) *redis.MapStringInterfaceCmd { //nolint:stylecheck // .
	return l.instance().FTCursorRead(ctx, index, cursorId, count)
}

func (l *lb) FTDictAdd(ctx context.Context, dict string, term ...any) *redis.IntCmd {
	return l.instance().FTDictAdd(ctx, dict, term...)
}

func (l *lb) FTDictDel(ctx context.Context, dict string, term ...any) *redis.IntCmd {
	return l.instance().FTDictDel(ctx, dict, term...)
}

func (l *lb) FTDictDump(ctx context.Context, dict string) *redis.StringSliceCmd {
	return l.instance().FTDictDump(ctx, dict)
}

func (l *lb) FTDropIndex(ctx context.Context, index string) *redis.StatusCmd {
	return l.instance().FTDropIndex(ctx, index)
}

func (l *lb) FTDropIndexWithArgs(ctx context.Context, index string, options *redis.FTDropIndexOptions) *redis.StatusCmd {
	return l.instance().FTDropIndexWithArgs(ctx, index, options)
}

func (l *lb) FTExplain(ctx context.Context, index, query string) *redis.StringCmd {
	return l.instance().FTExplain(ctx, index, query)
}

func (l *lb) FTExplainWithArgs(ctx context.Context, index, query string, options *redis.FTExplainOptions) *redis.StringCmd {
	return l.instance().FTExplainWithArgs(ctx, index, query, options)
}

func (l *lb) FTInfo(ctx context.Context, index string) *redis.FTInfoCmd {
	return l.instance().FTInfo(ctx, index)
}

func (l *lb) FTSpellCheck(ctx context.Context, index, query string) *redis.FTSpellCheckCmd {
	return l.instance().FTSpellCheck(ctx, index, query)
}

func (l *lb) FTSpellCheckWithArgs(ctx context.Context, index, query string, options *redis.FTSpellCheckOptions) *redis.FTSpellCheckCmd {
	return l.instance().FTSpellCheckWithArgs(ctx, index, query, options)
}

func (l *lb) FTSearch(ctx context.Context, index, query string) *redis.FTSearchCmd {
	return l.instance().FTSearch(ctx, index, query)
}

func (l *lb) FTSearchWithArgs(ctx context.Context, index, query string, options *redis.FTSearchOptions) *redis.FTSearchCmd {
	return l.instance().FTSearchWithArgs(ctx, index, query, options)
}

func (l *lb) FTSynDump(ctx context.Context, index string) *redis.FTSynDumpCmd {
	return l.instance().FTSynDump(ctx, index)
}

func (l *lb) FTSynUpdate(ctx context.Context, index string, synGroupId any, terms []any) *redis.StatusCmd { //nolint:stylecheck // .
	return l.instance().FTSynUpdate(ctx, index, synGroupId, terms)
}

//nolint:stylecheck // .
func (l *lb) FTSynUpdateWithArgs(ctx context.Context, index string, synGroupId any, options *redis.FTSynUpdateOptions, terms []any) *redis.StatusCmd {
	return l.instance().FTSynUpdateWithArgs(ctx, index, synGroupId, options, terms)
}

func (l *lb) FTTagVals(ctx context.Context, index, field string) *redis.StringSliceCmd {
	return l.instance().FTTagVals(ctx, index, field)
}

func (l *lb) ACLCat(ctx context.Context) *redis.StringSliceCmd {
	return l.instance().ACLCat(ctx)
}

func (l *lb) ACLCatArgs(ctx context.Context, options *redis.ACLCatArgs) *redis.StringSliceCmd {
	return l.instance().ACLCatArgs(ctx, options)
}

func (l *lb) ACLDelUser(ctx context.Context, username string) *redis.IntCmd {
	return l.instance().ACLDelUser(ctx, username)
}

func (l *lb) ACLList(ctx context.Context) *redis.StringSliceCmd {
	return l.instance().ACLList(ctx)
}

func (l *lb) ACLSetUser(ctx context.Context, username string, rules ...string) *redis.StatusCmd {
	return l.instance().ACLSetUser(ctx, username, rules...)
}

func (l *lb) ClusterMyID(ctx context.Context) *redis.StringCmd {
	return l.instance().ClusterMyID(ctx)
}

func (l *lb) HGetDel(ctx context.Context, key string, fields ...string) *redis.StringSliceCmd {
	return l.instance().HGetDel(ctx, key, fields...)
}

func (l *lb) HGetEX(ctx context.Context, key string, fields ...string) *redis.StringSliceCmd {
	return l.instance().HGetEX(ctx, key, fields...)
}

func (l *lb) HGetEXWithArgs(ctx context.Context, key string, options *redis.HGetEXOptions, fields ...string) *redis.StringSliceCmd {
	return l.instance().HGetEXWithArgs(ctx, key, options, fields...)
}

func (l *lb) HSetEX(ctx context.Context, key string, fieldsAndValues ...string) *redis.IntCmd {
	return l.instance().HSetEX(ctx, key, fieldsAndValues...)
}

func (l *lb) HSetEXWithArgs(ctx context.Context, key string, options *redis.HSetEXOptions, fieldsAndValues ...string) *redis.IntCmd {
	return l.instance().HSetEXWithArgs(ctx, key, options, fieldsAndValues...)
}

func (l *lb) HStrLen(ctx context.Context, key, field string) *redis.IntCmd {
	return l.instance().HStrLen(ctx, key, field)
}

func (l *lb) Close() error {
	wg := new(sync.WaitGroup)
	wg.Add(len(l.instances))
	errs := make(chan error, len(l.instances))
	for ix, instance := range l.instances {
		go func(ixx int, client *redis.Client) {
			defer wg.Done()
			errs <- errors.Wrapf(client.Close(), "failed to close instance %v", l.urls[ixx])
		}(ix, instance)
	}
	wg.Wait()
	close(errs)
	errs2 := make([]error, 0, len(l.instances))
	for err := range errs {
		errs2 = append(errs2, err)
	}

	return multierror.Append(nil, errs2...).ErrorOrNil() //nolint:wrapcheck // Not needed, they're the same.
}

func (l *lb) Ping(ctx context.Context) *redis.StatusCmd { //nolint:funlen // Not an issue.
	wg := new(sync.WaitGroup)
	wg.Add(len(l.instances))
	responses := make(chan *redis.StatusCmd, len(l.instances))
	for ix, instance := range l.instances {
		go func(ixx int, client *redis.Client) {
			defer wg.Done()
			res := client.Ping(ctx)
			res.SetVal(fmt.Sprintf("[%v]%v", l.urls[ixx], res.Val()))
			responses <- res
		}(ix, instance)
	}
	wg.Wait()
	close(responses)
	var failedOne, succeededOne *redis.StatusCmd
	errs2 := make([]error, 0, len(l.instances))
	for res := range responses {
		if res.Err() != nil {
			failedOne = res
			errs2 = append(errs2, errors.Wrapf(res.Err(), "%v", res.Val()))
		} else {
			succeededOne = res
		}
	}
	if failedOne != nil {
		failedOne.SetErr(multierror.Append(nil, errs2...).ErrorOrNil())

		return failedOne
	}
	succeededOne.SetVal("PONG")

	return succeededOne
}

//nolint:funlen,gocognit,revive // .
func (l *lb) IsRW(ctx context.Context) bool {
	wg := new(sync.WaitGroup)
	wg.Add(len(l.instances))
	errChan := make(chan error, len(l.instances))
	for ix, inst := range l.instances {
		go func(iix int, cl *redis.Client) {
			defer wg.Done()
			responses, err := cl.Pipelined(ctx, func(pipeliner redis.Pipeliner) error {
				k1 := fmt.Sprintf("rw-check-1-%v", uuid.NewString())
				k2 := fmt.Sprintf("rw-check-2-%v", uuid.NewString())
				now := *time.Now().Time
				if err := pipeliner.Set(ctx, k1, now, stdlibtime.Minute).Err(); err != nil {
					return err //nolint:wrapcheck // Not needed.
				}
				if err := pipeliner.Set(ctx, k2, now, stdlibtime.Minute).Err(); err != nil {
					return err //nolint:wrapcheck // Not needed.
				}

				return nil
			})
			if err == nil {
				errs := make([]error, 0, 1+1)
				for _, resp := range responses {
					errs = append(errs, resp.Err())
				}
				err = errors.Wrapf(multierror.Append(nil, errs...).ErrorOrNil(), "[%v]", l.urls[iix])
			}
			errChan <- errors.Wrapf(err, "[%v]", l.urls[iix])
		}(ix, inst)
	}
	wg.Wait()
	close(errChan)
	errs := make([]error, 0, len(l.instances))
	for err := range errChan {
		errs = append(errs, err)
	}
	err := multierror.Append(nil, errs...).ErrorOrNil()
	log.Error(errors.Wrap(err, "storage/v3 rw-check failed"))

	return err == nil
}

func (l *lb) instance() *redis.Client {
	return l.instances[atomic.AddUint64(&l.currentIndex, 1)%uint64(len(l.instances))]
}

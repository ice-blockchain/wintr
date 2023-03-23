#!/usr/bin/env tarantool
-- SPDX-License-Identifier: ice License 1.0
-- Details: https:www.tarantool.io/en/doc/latest/reference/configuration/
require "lfs"
lfs.mkdir("./memtx")
lfs.mkdir("./wal")

box.cfg{
    listen = %[1]v,
    work_dir = '.',
    wal_dir = './wal',
    memtx_dir = './memtx',
    memtx_memory = 1073741824, -- that`s 1Gb
    sql_cache_size = 67108864, -- that`s 64Mb
    memtx_use_mvcc_engine = true,
    pid_file = "tarantool.pid",
}
box.schema.user.passwd('pass') -- pass of the `admin` user

function get_cluster_members()
    a = {}
    a["uri"] = 'localhost:%[1]v' -- the ip/host and the port of the server that can be accessed from the client
    a["type"] = 'writable'

    return { a }
end

function get_all_user_spaces()
    r  = {}
    for name, s in pairs(box.space) do
    	if type(name) == 'string' and name:match("%%D") ~= nil and name:find("_") ~= 1 then
    	    e  = {}
    		for k,v in pairs(box.space[name]) do
                  if (type(v) ~= "function") then
                    e[k] = v
                  end
            end
    		r[name] = e
    	end
    end
    return r
end

function enable_sync_on_all_user_spaces()
    if box.cfg.election_mode ~= 'off' then
        for name in pairs(get_all_user_spaces()) do
            box.space[name]:alter({is_sync = true})
        end
    end
end
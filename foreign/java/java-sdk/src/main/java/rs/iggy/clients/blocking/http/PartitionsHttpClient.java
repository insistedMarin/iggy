/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package rs.iggy.clients.blocking.http;

import org.apache.hc.core5.http.message.BasicNameValuePair;
import rs.iggy.clients.blocking.PartitionsClient;
import rs.iggy.identifier.StreamId;
import rs.iggy.identifier.TopicId;

class PartitionsHttpClient implements PartitionsClient {

    private static final String STREAMS = "/streams";
    private static final String TOPICS = "/topics";
    private static final String PARTITIONS = "/partitions";
    private final InternalHttpClient httpClient;

    public PartitionsHttpClient(InternalHttpClient httpClient) {
        this.httpClient = httpClient;
    }

    @Override
    public void createPartitions(StreamId streamId, TopicId topicId, Long partitionsCount) {
        var request = httpClient.preparePostRequest(STREAMS + "/" + streamId + TOPICS + "/" + topicId + PARTITIONS,
                new CreatePartitions(partitionsCount));
        httpClient.execute(request);
    }

    @Override
    public void deletePartitions(StreamId streamId, TopicId topicId, Long partitionsCount) {
        var request = httpClient.prepareDeleteRequest(STREAMS + "/" + streamId + TOPICS + "/" + topicId + PARTITIONS,
                new BasicNameValuePair("partitions_count", partitionsCount.toString()));
        httpClient.execute(request);
    }

    private record CreatePartitions(Long partitionsCount) {
    }
}

# start / stop the cluster, or clean all logs and local LogEntries
start:
	docker-compose -f docker-compose-local.yaml up
stop:
	docker-compose -f docker-compose-local.yaml down
clean:
	@NodeNames=`ls | grep "^node[0-9]*"` && \
	for node in $$NodeNames; \
	do \
		rm -f $$node/var/*.log && rm -f $$node/var/*.db; \
	done

start:
	docker-compose -f docker-compose-local.yaml up
stop:
	docker-compose -f docker-compose-local.yaml down
clean:
	rm -f node1/var/*.log
	rm -f node2/var/*.log
	rm -f node3/var/*.log
	rm -f node1/var/*.db
	rm -f node2/var/*.db
	rm -f node3/var/*.db

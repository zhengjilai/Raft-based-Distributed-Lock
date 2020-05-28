start:
	docker-compose -f docker-compose-local.yaml up
stop:
	docker-compose -f docker-compose-local.yaml down
clean:
	rm node1/var/*.log
	rm node2/var/*.log
	rm node3/var/*.log
	rm node1/var/*.db
	rm node2/var/*.db
	rm node3/var/*.db
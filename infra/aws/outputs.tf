output "server_public_ip" { value = aws_instance.server.public_ip }
output "server_private_ip" { value = aws_instance.server.private_ip }
output "generator_public_ip" { value = aws_instance.generator.public_ip }
output "ssh_server" { value = "ssh ubuntu@${aws_instance.server.public_ip}" }
output "ssh_generator" { value = "ssh ubuntu@${aws_instance.generator.public_ip}" }
output "k6_host_hint" { value = "na generatorze: SERVER=${aws_instance.server.private_ip} (porty 8443/8444/8445)" }
